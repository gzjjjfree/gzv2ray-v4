package pipe

import (
	"errors"
	"io"
	"runtime"
	"sync"
	"time"
	//"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/buf"
	"github.com/gzjjjfree/gzv2ray-v4/common/signal"
	"github.com/gzjjjfree/gzv2ray-v4/common/signal/done"
)

type state byte

const (
	open state = iota
	closed
	errord
)

type pipeOption struct {
	limit           int32 // maximum buffer size in bytes
	discardOverflow bool
}

func (o *pipeOption) isFull(curSize int32) bool {
	return o.limit >= 0 && curSize > o.limit
}

type pipe struct {
	sync.Mutex
	data        buf.MultiBuffer
	readSignal  *signal.Notifier
	writeSignal *signal.Notifier
	done        *done.Instance
	option      pipeOption
	state       state
}

var errBufferFull = errors.New("buffer full")
var errSlowDown = errors.New("slow down")

func (p *pipe) getState(forRead bool) error {
	//fmt.Println("in transport-pipe-impl.go func (p *pipe) getState")
	switch p.state {
	case open:
		if !forRead && p.option.isFull(p.data.Len()) {
			return errBufferFull
		}
		return nil
	case closed:
		if !forRead {
			return io.ErrClosedPipe
		}
		if !p.data.IsEmpty() {
			return nil
		}
		return io.EOF
	case errord:
		return io.ErrClosedPipe
	default:
		panic("impossible case")
	}
}

func (p *pipe) readMultiBufferInternal() (buf.MultiBuffer, error) {
	//fmt.Println("in transport-pipe-impl.go func (p *pipe) readMultiBufferInternal()")
	p.Lock()
	defer p.Unlock()

	if err := p.getState(true); err != nil {
		return nil, err
	}

	data := p.data
	p.data = nil
	return data, nil
}

func (p *pipe) ReadMultiBuffer() (buf.MultiBuffer, error) {
	//fmt.Println("in transport-pipe-impl.go func (p *pipe) ReadMultiBuffer()")
	for {
		data, err := p.readMultiBufferInternal()
		
		if data != nil || err != nil {
			 
			//fmt.Println("in transport-pipe-impl.go func (p *pipe) ReadMultiBuffer() data != nil")

			p.writeSignal.Signal()
			return data, err
		}
		//fmt.Println("in transport-pipe-impl.go func (p *pipe) ReadMultiBuffer() wait data")
		select {
		case <-p.readSignal.Wait()://fmt.Println("in transport-pipe-impl.go func (p *pipe) ReadMultiBuffer() wait data <-p.readSignal.Wait()")
		case <-p.done.Wait()://fmt.Println("in transport-pipe-impl.go func (p *pipe) ReadMultiBuffer() wait data <-p.done.Wait()")
		}
	}
}

func (p *pipe) ReadMultiBufferTimeout(d time.Duration) (buf.MultiBuffer, error) {
	//fmt.Println("in transport-pipe-impl.go func (p *pipe) ReadMultiBufferTimeout")
	timer := time.NewTimer(d)
	defer timer.Stop()

	for {
		data, err := p.readMultiBufferInternal()
		//fmt.Println("in transport-pipe-impl.go func (p *pipe) ReadMultiBufferTimeout data: ")
		if data != nil || err != nil {
			
			 
			//fmt.Println("in transport-pipe-impl.go func (p *pipe) ReadMultiBufferTimeout data != nil")

			p.writeSignal.Signal()
		    return data, err
		    }

		select {
		case <-p.readSignal.Wait():
		case <-p.done.Wait():
		case <-timer.C:
			return nil, buf.ErrReadTimeout
		}
	}
}

func (p *pipe) writeMultiBufferInternal(mb buf.MultiBuffer) error {
	//fmt.Println("in transport-pipe-impl.go func (p *pipe) writeMultiBufferInternal")
	p.Lock()
	defer p.Unlock()

	if err := p.getState(false); err != nil {
		return err
	}

	if p.data == nil {
		p.data = mb
		return nil
	}

	p.data, _ = buf.MergeMulti(p.data, mb)
	return errSlowDown
}

func (p *pipe) WriteMultiBuffer(mb buf.MultiBuffer) error {
	//fmt.Println("in transport-pipe-impl.go func (p *pipe) WriteMultiBuffer")
	if mb.IsEmpty() {
		return nil
	}

	for {
		err := p.writeMultiBufferInternal(mb)
		if err == nil {
			p.readSignal.Signal()
			return nil
		}

		if err == errSlowDown {
			p.readSignal.Signal()

			// Yield current goroutine. Hopefully the reading counterpart can pick up the payload.
			runtime.Gosched()
			return nil
		}

		if err == errBufferFull && p.option.discardOverflow {
			buf.ReleaseMulti(mb)
			return nil
		}

		if err != errBufferFull {
			buf.ReleaseMulti(mb)
			p.readSignal.Signal()
			return err
		}

		select {
		case <-p.writeSignal.Wait():
		case <-p.done.Wait():
			return io.ErrClosedPipe
		}
	}
}

func (p *pipe) Close() error {
	//fmt.Println("in transport-pipe-impl.go func (p *pipe) Close()")
	p.Lock()
	defer p.Unlock()

	if p.state == closed || p.state == errord {
		return nil
	}

	p.state = closed
	common.Must(p.done.Close())
	return nil
}

// Interrupt implements common.Interruptible.
func (p *pipe) Interrupt() {
	p.Lock()
	defer p.Unlock()

	if p.state == closed || p.state == errord {
		return
	}

	p.state = errord

	if !p.data.IsEmpty() {
		buf.ReleaseMulti(p.data)
		p.data = nil
	}

	common.Must(p.done.Close())
}
