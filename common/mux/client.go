package mux

import (
	"context"
	"io"
	"sync"
	"time"
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/buf"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/protocol"
	"github.com/gzjjjfree/gzv2ray-v4/common/session"
	"github.com/gzjjjfree/gzv2ray-v4/common/signal/done"
	"github.com/gzjjjfree/gzv2ray-v4/common/task"
	"github.com/gzjjjfree/gzv2ray-v4/proxy"
	"github.com/gzjjjfree/gzv2ray-v4/transport"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
	"github.com/gzjjjfree/gzv2ray-v4/transport/pipe"
	"github.com/gzjjjfree/gzv2ray-v4/common/serial"
)


var muxCoolAddress = net.DomainAddress("v1.mux.cool")

type ClientManager struct {
	Enabled bool // wheather mux is enabled from user config
	Picker  WorkerPicker
}

func (m *ClientManager) Dispatch(ctx context.Context, link *transport.Link) error {
	for i := 0; i < 16; i++ {
		worker, err := m.Picker.PickAvailable()
		if err != nil {
			return err
		}
		if worker.Dispatch(ctx, link) {
			return nil
		}
	}

	return errors.New("unable to find an available mux client")
}

type WorkerPicker interface {
	PickAvailable() (*ClientWorker, error)
}

type IncrementalWorkerPicker struct {
	Factory ClientWorkerFactory

	access      sync.Mutex
	workers     []*ClientWorker
	cleanupTask *task.Periodic
}

func (p *IncrementalWorkerPicker) cleanupFunc() error {
	p.access.Lock()
	defer p.access.Unlock()

	if len(p.workers) == 0 {
		return errors.New("no worker")
	}

	p.cleanup()
	return nil
}

func (p *IncrementalWorkerPicker) cleanup() {
	var activeWorkers []*ClientWorker
	for _, w := range p.workers {
		if !w.Closed() {
			activeWorkers = append(activeWorkers, w)
		}
	}
	p.workers = activeWorkers
}

func (p *IncrementalWorkerPicker) findAvailable() int {
	for idx, w := range p.workers {
		if !w.IsFull() {
			return idx
		}
	}

	return -1
}

func (p *IncrementalWorkerPicker) pickInternal() (*ClientWorker, bool, error) {
	p.access.Lock()
	defer p.access.Unlock()

	idx := p.findAvailable()
	if idx >= 0 {
		n := len(p.workers)
		if n > 1 && idx != n-1 {
			p.workers[n-1], p.workers[idx] = p.workers[idx], p.workers[n-1]
		}
		return p.workers[idx], false, nil
	}

	p.cleanup()

	worker, err := p.Factory.Create()
	if err != nil {
		return nil, false, err
	}
	p.workers = append(p.workers, worker)

	if p.cleanupTask == nil {
		p.cleanupTask = &task.Periodic{
			Interval: time.Second * 30,
			Execute:  p.cleanupFunc,
		}
	}

	return worker, true, nil
}

func (p *IncrementalWorkerPicker) PickAvailable() (*ClientWorker, error) {
	worker, start, err := p.pickInternal()
	if start {
		common.Must(p.cleanupTask.Start())
	}

	return worker, err
}

type ClientWorkerFactory interface {
	Create() (*ClientWorker, error)
}

type DialingWorkerFactory struct {
	Proxy    proxy.Outbound
	Dialer   internet.Dialer
	Strategy ClientStrategy

	ctx context.Context
}

func NewDialingWorkerFactory(ctx context.Context, proxy proxy.Outbound, dialer internet.Dialer, strategy ClientStrategy) *DialingWorkerFactory {
	return &DialingWorkerFactory{
		Proxy:    proxy,
		Dialer:   dialer,
		Strategy: strategy,
		ctx:      ctx,
	}
}
func (f *DialingWorkerFactory) Create() (*ClientWorker, error) {
	fmt.Println("in common-mux-client.go func  (f *DialingWorkerFactory) Create()")
	opts := []pipe.Option{pipe.WithSizeLimit(64 * 1024)}
	uplinkReader, upLinkWriter := pipe.New(opts...)
	downlinkReader, downlinkWriter := pipe.New(opts...)
	
	c, err := NewClientWorker(transport.Link{
		Reader: downlinkReader,
		Writer: upLinkWriter,
	}, f.Strategy)
	
	if err != nil {
		return nil, err
	}

	go func(p proxy.Outbound, d internet.Dialer, c common.Closable) {
		fmt.Println("in common-mux-client.go func  (f *DialingWorkerFactory) Create() go func")
		ctx := session.ContextWithOutbound(f.ctx, &session.Outbound{
			Target: net.TCPDestination(muxCoolAddress, muxCoolPort),
		})
		ctx, cancel := context.WithCancel(ctx)
		
		if err := p.Process(ctx, &transport.Link{Reader: uplinkReader, Writer: downlinkWriter}, d); err != nil {
			//fmt.Println("failed to handler mux client connection err: ", err)
		}
		fmt.Println("in common-mux-client.go func  (f *DialingWorkerFactory) Create()  common.Must(c.Close())")
		common.Must(c.Close())
		cancel()
	}(f.Proxy, f.Dialer, c.done)

	return c, nil
}

type ClientStrategy struct {
	MaxConcurrency uint32
	MaxConnection  uint32
}

type ClientWorker struct {
	sessionManager *SessionManager
	link           transport.Link
	done           *done.Instance
	strategy       ClientStrategy
}


var muxCoolPort = net.Port(8527)

// NewClientWorker creates a new mux.Client.
func NewClientWorker(stream transport.Link, s ClientStrategy) (*ClientWorker, error) {
	c := &ClientWorker{
		sessionManager: NewSessionManager(),
		link:           stream,
		done:           done.New(),
		strategy:       s,
	}

	go c.fetchOutput()
	go c.monitor()

	return c, nil
}

func (m *ClientWorker) TotalConnections() uint32 {
	return uint32(m.sessionManager.Count())
}

func (m *ClientWorker) ActiveConnections() uint32 {
	return uint32(m.sessionManager.Size())
}

// Closed returns true if this Client is closed.
func (m *ClientWorker) Closed() bool {
	return m.done.Done()
}

func (m *ClientWorker) monitor() {
	timer := time.NewTicker(time.Second * 16)
	defer timer.Stop()

	for {
		select {
		case <-m.done.Wait():
			m.sessionManager.Close()
			common.Close(m.link.Writer)
			common.Interrupt(m.link.Reader)
			return
		case <-timer.C:
			size := m.sessionManager.Size()
			if size == 0 && m.sessionManager.CloseIfNoSession() {
				common.Must(m.done.Close())
			}
		}
	}
}

func writeFirstPayload(reader buf.Reader, writer *Writer) error {
	fmt.Println("in common-mux-client.go func writeFirstPayload")
	err := buf.CopyOnceTimeout(reader, writer, time.Millisecond*100)
	if err == buf.ErrNotTimeoutReader || err == buf.ErrReadTimeout {
		return writer.WriteMultiBuffer(buf.MultiBuffer{})
	}

	if err != nil {
		return err
	}

	return nil
}

func fetchInput(ctx context.Context, s *Session, output buf.Writer) {
	fmt.Println("in common-mux-client.go func fetchInput")
	dest := session.OutboundFromContext(ctx).Target
	transferType := protocol.TransferTypeStream
	if dest.Network == net.Network_UDP {
		transferType = protocol.TransferTypePacket
	}
	s.transferType = transferType
	writer := NewWriter(s.ID, dest, output, transferType)
	defer s.Close()
	defer writer.Close()

	//fmt.Println("dispatching request to ")
	if err := writeFirstPayload(s.input, writer); err != nil {
		fmt.Println("failed to write first payload")
		writer.hasError = true
		common.Interrupt(s.input)
		return
	}
	//for i:=0;i<100000;i++{errors.New("ok")}
	fmt.Println("in common-mux-client.go func fetchInput buf.Copy")
	if err := buf.Copy(s.input, writer); err != nil {
		//fmt.Println("failed to fetch all input")
		writer.hasError = true
		common.Interrupt(s.input)
		return
	}
}

func (m *ClientWorker) IsClosing() bool {
	sm := m.sessionManager
	if m.strategy.MaxConnection > 0 && sm.Count() >= int(m.strategy.MaxConnection) {
		return true
	}
	return false
}

func (m *ClientWorker) IsFull() bool {
	if m.IsClosing() || m.Closed() {
		return true
	}

	sm := m.sessionManager
	if m.strategy.MaxConcurrency > 0 && sm.Size() >= int(m.strategy.MaxConcurrency) {
		return true
	}
	return false
}

func (m *ClientWorker) Dispatch(ctx context.Context, link *transport.Link) bool {
	fmt.Println("in common-mux-client.go func (m *ClientWorker) Dispatch")
	if m.IsFull() || m.Closed() {
		return false
	}

	sm := m.sessionManager
	s := sm.Allocate()
	if s == nil {
		return false
	}
	s.input = link.Reader
	s.output = link.Writer
	go fetchInput(ctx, s, m.link.Writer)
	return true
}

func (m *ClientWorker) handleStatueKeepAlive(meta *FrameMetadata, reader *buf.BufferedReader) error {
	fmt.Println("in common-mux-client.go func (m *ClientWorker) handleStatueKeepAlive")
	if meta.Option.Has(OptionData) {
		return buf.Copy(NewStreamReader(reader), buf.Discard)
	}
	return nil
}

func (m *ClientWorker) handleStatusNew(meta *FrameMetadata, reader *buf.BufferedReader) error {
	fmt.Println("in common-mux-client.go func (m *ClientWorker) handleStatusNew meta.Option.Has(OptionData): ", meta.Option.Has(OptionData))
	if meta.Option.Has(OptionData) {
		return buf.Copy(NewStreamReader(reader), buf.Discard)
	}
	return nil
}

func (m *ClientWorker) handleStatusKeep(meta *FrameMetadata, reader *buf.BufferedReader) error {
	if !meta.Option.Has(OptionData) {
		return nil
	}

	s, found := m.sessionManager.Get(meta.SessionID)
	if !found {
		// Notify remote peer to close this session.
		closingWriter := NewResponseWriter(meta.SessionID, m.link.Writer, protocol.TransferTypeStream)
		closingWriter.Close()

		return buf.Copy(NewStreamReader(reader), buf.Discard)
	}

	rr := s.NewReader(reader)
	err := buf.Copy(rr, s.output)
	if err != nil && buf.IsWriteError(err) {
		fmt.Println("failed to write to downstream. closing session ")

		// Notify remote peer to close this session.
		closingWriter := NewResponseWriter(meta.SessionID, m.link.Writer, protocol.TransferTypeStream)
		closingWriter.Close()

		drainErr := buf.Copy(rr, buf.Discard)
		common.Interrupt(s.input)
		s.Close()
		return drainErr
	}

	return err
}

func (m *ClientWorker) handleStatusEnd(meta *FrameMetadata, reader *buf.BufferedReader) error {
	if s, found := m.sessionManager.Get(meta.SessionID); found {
		if meta.Option.Has(OptionError) {
			common.Interrupt(s.input)
			common.Interrupt(s.output)
		}
		s.Close()
	}
	if meta.Option.Has(OptionData) {
		return buf.Copy(NewStreamReader(reader), buf.Discard)
	}
	return nil
}

func (m *ClientWorker) fetchOutput() {
	defer func() {
		common.Must(m.done.Close())
	}()

	reader := &buf.BufferedReader{Reader: m.link.Reader}

	var meta FrameMetadata
	for {
		err := meta.Unmarshal(reader)
		if err != nil {
			fmt.Println("in common-mux-clien.go func (m *ClientWorker) fetchOutput() meta.Unmarshal(reader) err: ", err)
			if serial.ToString(err) != serial.ToString(io.EOF) && serial.ToString(err)[:2] != "io" && serial.ToString(err)[:2] != "re" {
				fmt.Println("failed to read metadata")
			}
			break
		}
		//fmt.Println("in common-mux-clien.go func (m *ClientWorker) fetchOutput() meta.SessionStatus: ", meta.SessionStatus)
		switch meta.SessionStatus {
		case SessionStatusKeepAlive:
			err = m.handleStatueKeepAlive(&meta, reader)
		case SessionStatusEnd:
			err = m.handleStatusEnd(&meta, reader)
		case SessionStatusNew:
			err = m.handleStatusNew(&meta, reader)
		case SessionStatusKeep:
			err = m.handleStatusKeep(&meta, reader)
		default:
			status := meta.SessionStatus
			fmt.Println("unknown status: ", status)
			return
		}

		if err != nil {
			fmt.Println("in common-mux-clien.go func (m *ClientWorker) fetchOutput() failed to process data")
		}
	}
}
