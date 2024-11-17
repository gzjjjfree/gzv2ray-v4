package inbound

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
	"errors"
	"fmt"

	"example.com/gztest"

	"github.com/gzjjjfree/gzv2ray-v4/app/proxyman"
	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/buf"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	//"github.com/gzjjjfree/gzv2ray-v4/common/serial"
	"github.com/gzjjjfree/gzv2ray-v4/common/session"
	"github.com/gzjjjfree/gzv2ray-v4/common/signal/done"
	"github.com/gzjjjfree/gzv2ray-v4/common/task"
	"github.com/gzjjjfree/gzv2ray-v4/features/routing"
	"github.com/gzjjjfree/gzv2ray-v4/features/stats"
	"github.com/gzjjjfree/gzv2ray-v4/proxy"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/tcp"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/udp"
	"github.com/gzjjjfree/gzv2ray-v4/transport/pipe"
)

type worker interface {
	Start() error
	Close() error
	Port() net.Port
	Proxy() proxy.Inbound
}

type tcpWorker struct {
	address         net.Address
	port            net.Port
	proxy           proxy.Inbound
	stream          *internet.MemoryStreamConfig
	recvOrigDest    bool
	tag             string
	dispatcher      routing.Dispatcher
	sniffingConfig  *proxyman.SniffingConfig
	uplinkCounter   stats.Counter
	downlinkCounter stats.Counter

	hub internet.Listener

	ctx context.Context
}

func getTProxyType(s *internet.MemoryStreamConfig) internet.SocketConfig_TProxyMode {
	if s == nil || s.SocketSettings == nil {
		return internet.SocketConfig_Off
	}
	return s.SocketSettings.Tproxy
}

func (w *tcpWorker) callback(conn internet.Connection) {
	fmt.Println("in  app-proxyman-inbound-worker.go func (w *tcpWorker) callback")
	ctx, cancel := context.WithCancel(w.ctx)
	sid := session.NewID()
	ctx = session.ContextWithID(ctx, sid)

	if w.recvOrigDest {
		var dest net.Destination
		fmt.Println("getTProxyType(w.stream) is: ")
		gztest.GetMessageReflectType(getTProxyType(w.stream))
		switch getTProxyType(w.stream) {
		case internet.SocketConfig_Redirect:
			d, err := tcp.GetOriginalDestination(conn)
			if err != nil {
				errors.New("failed to get original destination")
			} else {
				dest = d
			}
		case internet.SocketConfig_TProxy:
			dest = net.DestinationFromAddr(conn.LocalAddr())
		}
		if dest.IsValid() {
			ctx = session.ContextWithOutbound(ctx, &session.Outbound{
				Target: dest,
			})
		}
	}
	ctx = session.ContextWithInbound(ctx, &session.Inbound{
		Source:  net.DestinationFromAddr(conn.RemoteAddr()),
		Gateway: net.TCPDestination(w.address, w.port),
		Tag:     w.tag,
	})
	content := new(session.Content)
	if w.sniffingConfig != nil {
		fmt.Println("in  app-proxyman-inbound-worker.go func (w *tcpWorker) callback w.sniffingConfig != nil")
		content.SniffingRequest.Enabled = w.sniffingConfig.Enabled
		content.SniffingRequest.OverrideDestinationForProtocol = w.sniffingConfig.DestinationOverride
		content.SniffingRequest.MetadataOnly = w.sniffingConfig.MetadataOnly
	}
	ctx = session.ContextWithContent(ctx, content)
	if w.uplinkCounter != nil || w.downlinkCounter != nil {
		fmt.Println("in  app-proxyman-inbound-worker.go func (w *tcpWorker) callback w.uplinkCounter != nil || w.downlinkCounter != nil")
		conn = &internet.StatCouterConnection{
			Connection:   conn,
			ReadCounter:  w.uplinkCounter,
			WriteCounter: w.downlinkCounter,
		}
	}
	if err := w.proxy.Process(ctx, net.Network_TCP, conn, w.dispatcher); err != nil {
		errors.New("connection ends")
	}
	cancel()
	if err := conn.Close(); err != nil {
		errors.New("failed to close connection")
	}
}

func (w *tcpWorker) Proxy() proxy.Inbound {
	return w.proxy
}

func (w *tcpWorker) Start() error {
	fmt.Println("in  app-proxyman-inbound-worker.go func (w *tcpWorker) Start()")
	ctx := context.Background()
	hub, err := internet.ListenTCP(ctx, w.address, w.port, w.stream, func(conn internet.Connection) {
		fmt.Println("in  app-proxyman-inbound-worker.go func (w *tcpWorker) Start() 开启一个协程 *tcpWorker-callback 等待数据")
		// 开启一个协程等待数据
		go w.callback(conn)
	})
	if err != nil {
		return errors.New("failed to listen TCP on ")
	}
	w.hub = hub
	return nil
}

func (w *tcpWorker) Close() error {
	fmt.Println("in app-proxyman-inbound-worker.go func (w *tcpWorker) Close()")
	var errorsmsg []interface{}
	if w.hub != nil {
		if err := common.Close(w.hub); err != nil {
			errorsmsg = append(errorsmsg, err)
		}
		if err := common.Close(w.proxy); err != nil {
			errorsmsg = append(errorsmsg, err)
		}
	}
	if len(errorsmsg) > 0 {
		return errors.New("failed to close all resources")
	}

	return nil
}

func (w *tcpWorker) Port() net.Port {
	return w.port
}

type udpConn struct {
	lastActivityTime int64 // in seconds
	reader           buf.Reader
	writer           buf.Writer
	output           func([]byte) (int, error)
	remote           net.Addr
	local            net.Addr
	done             *done.Instance
	uplink           stats.Counter
	downlink         stats.Counter
}

func (c *udpConn) updateActivity() {
	atomic.StoreInt64(&c.lastActivityTime, time.Now().Unix())
}

// ReadMultiBuffer implements buf.Reader
func (c *udpConn) ReadMultiBuffer() (buf.MultiBuffer, error) {
	fmt.Println("in  app-proxyman-inbound-worker.go func (c *udpConn) ReadMultiBuffer()")
	mb, err := c.reader.ReadMultiBuffer()
	if err != nil {
		return nil, err
	}
	c.updateActivity()

	if c.uplink != nil {
		c.uplink.Add(int64(mb.Len()))
	}

	return mb, nil
}

func (c *udpConn) Read(buf []byte) (int, error) {
	panic("not implemented")
}

// Write implements io.Writer.
func (c *udpConn) Write(buf []byte) (int, error) {
	n, err := c.output(buf)
	if c.downlink != nil {
		c.downlink.Add(int64(n))
	}
	if err == nil {
		c.updateActivity()
	}
	return n, err
}

func (c *udpConn) Close() error {
	fmt.Println("in app-proxyman-inbound-worker.go func  (c *udpConn) Close() ")
	common.Must(c.done.Close())
	common.Must(common.Close(c.writer))
	return nil
}

func (c *udpConn) RemoteAddr() net.Addr {
	return c.remote
}

func (c *udpConn) LocalAddr() net.Addr {
	return c.local
}

func (*udpConn) SetDeadline(time.Time) error {
	return nil
}

func (*udpConn) SetReadDeadline(time.Time) error {
	return nil
}

func (*udpConn) SetWriteDeadline(time.Time) error {
	return nil
}

type connID struct {
	src  net.Destination
	dest net.Destination
}

type udpWorker struct {
	sync.RWMutex

	proxy           proxy.Inbound
	hub             *udp.Hub
	address         net.Address
	port            net.Port
	tag             string
	stream          *internet.MemoryStreamConfig
	dispatcher      routing.Dispatcher
	sniffingConfig  *proxyman.SniffingConfig
	uplinkCounter   stats.Counter
	downlinkCounter stats.Counter

	checker    *task.Periodic
	activeConn map[connID]*udpConn

	ctx context.Context
}

func (w *udpWorker) getConnection(id connID) (*udpConn, bool) {
	fmt.Println("in  app-proxyman-inbound-worker.go func (w *udpWorker) getConnection")
	w.Lock()
	defer w.Unlock()

	if conn, found := w.activeConn[id]; found && !conn.done.Done() {
		return conn, true
	}

	pReader, pWriter := pipe.New(pipe.DiscardOverflow(), pipe.WithSizeLimit(16*1024))
	conn := &udpConn{
		reader: pReader,
		writer: pWriter,
		output: func(b []byte) (int, error) {
			return w.hub.WriteTo(b, id.src)
		},
		remote: &net.UDPAddr{
			IP:   id.src.Address.IP(),
			Port: int(id.src.Port),
		},
		local: &net.UDPAddr{
			IP:   w.address.IP(),
			Port: int(w.port),
		},
		done:     done.New(),
		uplink:   w.uplinkCounter,
		downlink: w.downlinkCounter,
	}
	w.activeConn[id] = conn

	conn.updateActivity()
	return conn, false
}

func (w *udpWorker) callback(b *buf.Buffer, source net.Destination, originalDest net.Destination) {
	fmt.Println("in  app-proxyman-inbound-worker.go func (w *udpWorker) callback")
	id := connID{
		src: source,
	}
	if originalDest.IsValid() {
		id.dest = originalDest
	}
	conn, existing := w.getConnection(id)

	// payload will be discarded in pipe is full.
	conn.writer.WriteMultiBuffer(buf.MultiBuffer{b})

	if !existing {
		common.Must(w.checker.Start())

		go func() {
			ctx := w.ctx
			sid := session.NewID()
			ctx = session.ContextWithID(ctx, sid)

			if originalDest.IsValid() {
				ctx = session.ContextWithOutbound(ctx, &session.Outbound{
					Target: originalDest,
				})
			}
			ctx = session.ContextWithInbound(ctx, &session.Inbound{
				Source:  source,
				Gateway: net.UDPDestination(w.address, w.port),
				Tag:     w.tag,
			})
			content := new(session.Content)
			if w.sniffingConfig != nil {
				content.SniffingRequest.Enabled = w.sniffingConfig.Enabled
				content.SniffingRequest.OverrideDestinationForProtocol = w.sniffingConfig.DestinationOverride
				content.SniffingRequest.MetadataOnly = w.sniffingConfig.MetadataOnly
			}
			ctx = session.ContextWithContent(ctx, content)
			if err := w.proxy.Process(ctx, net.Network_UDP, conn, w.dispatcher); err != nil {
				errors.New("connection ends")
			}
			conn.Close()
			w.removeConn(id)
		}()
	}
}

func (w *udpWorker) removeConn(id connID) {
	w.Lock()
	delete(w.activeConn, id)
	w.Unlock()
}

func (w *udpWorker) handlePackets() {
	fmt.Println("in  app-proxyman-inbound-worker.go func (w *udpWorker) handlePackets()")
	receive := w.hub.Receive()
	for payload := range receive {
		w.callback(payload.Payload, payload.Source, payload.Target)
	}
}

func (w *udpWorker) clean() error {
	fmt.Println("in  app-proxyman-inbound-worker.go func (w *udpWorker) clean")
	nowSec := time.Now().Unix()
	w.Lock()
	defer w.Unlock()

	if len(w.activeConn) == 0 {
		return errors.New("no more connections. stopping...")
	}

	for addr, conn := range w.activeConn {
		if nowSec-atomic.LoadInt64(&conn.lastActivityTime) > 8 { // TODO Timeout too small
			delete(w.activeConn, addr)
			conn.Close()
		}
	}

	if len(w.activeConn) == 0 {
		w.activeConn = make(map[connID]*udpConn, 16)
	}

	return nil
}

func (w *udpWorker) Start() error {
	fmt.Println("in  app-proxyman-inbound-worker.go func (w *udpWorker) Start()")
	w.activeConn = make(map[connID]*udpConn, 16)
	ctx := context.Background()
	h, err := udp.ListenUDP(ctx, w.address, w.port, w.stream, udp.HubCapacity(256))
	if err != nil {
		return err
	}

	w.checker = &task.Periodic{
		Interval: time.Second * 16,
		Execute:  w.clean,
	}

	w.hub = h
	go w.handlePackets()
	return nil
}

func (w *udpWorker) Close() error {
	fmt.Println("in app-proxyman-inbound-worker.go func (w *udpWorker) Close() ")
	w.Lock()
	defer w.Unlock()

	var errorsmsg []interface{}

	if w.hub != nil {
		if err := w.hub.Close(); err != nil {
			errorsmsg = append(errorsmsg, err)
		}
	}

	if w.checker != nil {
		if err := w.checker.Close(); err != nil {
			errorsmsg = append(errorsmsg, err)
		}
	}

	if err := common.Close(w.proxy); err != nil {
		errorsmsg = append(errorsmsg, err)
	}

	if len(errorsmsg) > 0 {
		return errors.New("failed to close all resources")
	}
	return nil
}

func (w *udpWorker) Port() net.Port {
	return w.port
}

func (w *udpWorker) Proxy() proxy.Inbound {
	return w.proxy
}

type dsWorker struct {
	address         net.Address
	proxy           proxy.Inbound
	stream          *internet.MemoryStreamConfig
	tag             string
	dispatcher      routing.Dispatcher
	sniffingConfig  *proxyman.SniffingConfig
	uplinkCounter   stats.Counter
	downlinkCounter stats.Counter

	hub internet.Listener

	ctx context.Context
}

func (w *dsWorker) callback(conn internet.Connection) {
	fmt.Println("in  app-proxyman-inbound-worker.go func   (w *dsWorker) callback")
	ctx, cancel := context.WithCancel(w.ctx)
	sid := session.NewID()
	ctx = session.ContextWithID(ctx, sid)

	ctx = session.ContextWithInbound(ctx, &session.Inbound{
		Source:  net.DestinationFromAddr(conn.RemoteAddr()),
		Gateway: net.UnixDestination(w.address),
		Tag:     w.tag,
	})
	content := new(session.Content)
	if w.sniffingConfig != nil {
		content.SniffingRequest.Enabled = w.sniffingConfig.Enabled
		content.SniffingRequest.OverrideDestinationForProtocol = w.sniffingConfig.DestinationOverride
		content.SniffingRequest.MetadataOnly = w.sniffingConfig.MetadataOnly
	}
	ctx = session.ContextWithContent(ctx, content)
	if w.uplinkCounter != nil || w.downlinkCounter != nil {
		conn = &internet.StatCouterConnection{
			Connection:   conn,
			ReadCounter:  w.uplinkCounter,
			WriteCounter: w.downlinkCounter,
		}
	}
	if err := w.proxy.Process(ctx, net.Network_UNIX, conn, w.dispatcher); err != nil {
		errors.New("connection ends")
	}
	cancel()
	if err := conn.Close(); err != nil {
		errors.New("failed to close connection")
	}
}

func (w *dsWorker) Proxy() proxy.Inbound {
	return w.proxy
}

func (w *dsWorker) Port() net.Port {
	return net.Port(0)
}
func (w *dsWorker) Start() error {
	fmt.Println("in  app-proxyman-inbound-worker.go func  (w *dsWorker) Start()")
	ctx := context.Background()
	hub, err := internet.ListenUnix(ctx, w.address, w.stream, func(conn internet.Connection) {
		go w.callback(conn)
	})
	if err != nil {
		return errors.New("failed to listen Unix Domain Socket on ")
	}
	w.hub = hub
	return nil
}

func (w *dsWorker) Close() error {
	fmt.Println("in app-proxyman-inbound-worker.go func  (w *dsWorker) Close()  ")
	var errorsmsg []interface{}
	if w.hub != nil {
		if err := common.Close(w.hub); err != nil {
			errorsmsg = append(errorsmsg, err)
		}
		if err := common.Close(w.proxy); err != nil {
			errorsmsg = append(errorsmsg, err)
		}
	}
	if len(errorsmsg) > 0 {
		return errors.New("failed to close all resources")
	}

	return nil
}
