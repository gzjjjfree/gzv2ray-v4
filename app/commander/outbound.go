// +build !confonly

package commander

import (
	"context"
	"sync"
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/signal/done"
	"github.com/gzjjjfree/gzv2ray-v4/transport"
)

// OutboundListener is a net.Listener for listening gRPC connections.
type OutboundListener struct {
	buffer chan net.Conn
	done   *done.Instance
}

func (l *OutboundListener) add(conn net.Conn) {
	select {
	case l.buffer <- conn:
	case <-l.done.Wait():
		conn.Close()
	default:
		conn.Close()
	}
}

// Accept implements net.Listener.
func (l *OutboundListener) Accept() (net.Conn, error) {
	select {
	case <-l.done.Wait():
		return nil, errors.New("listen closed")
	case c := <-l.buffer:
		return c, nil
	}
}

// Close implement net.Listener.
func (l *OutboundListener) Close() error {
	fmt.Println("in app-commander-outbound.go func (l *OutboundListener) Close()")
	common.Must(l.done.Close())
L:
	for {
		select {
		case c := <-l.buffer:
			c.Close()
		default:
			break L
		}
	}
	return nil
}

// Addr implements net.Listener.
func (l *OutboundListener) Addr() net.Addr {
	return &net.TCPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: 0,
	}
}

// Outbound is a outbound.Handler that handles gRPC connections.
type Outbound struct {
	tag      string
	listener *OutboundListener
	access   sync.RWMutex
	closed   bool
}

// Dispatch implements outbound.Handler.
func (co *Outbound) Dispatch(ctx context.Context, link *transport.Link) {
	co.access.RLock()

	if co.closed {
		common.Interrupt(link.Reader)
		common.Interrupt(link.Writer)
		co.access.RUnlock()
		return
	}

	closeSignal := done.New()
	c := net.NewConnection(net.ConnectionInputMulti(link.Writer), net.ConnectionOutputMulti(link.Reader), net.ConnectionOnClose(closeSignal))
	co.listener.add(c)
	co.access.RUnlock()
	<-closeSignal.Wait()
}

// Tag implements outbound.Handler.
func (co *Outbound) Tag() string {
	return co.tag
}

// Start implements common.Runnable.
func (co *Outbound) Start() error {
	fmt.Println("in app-commander-outbound.go  func Start()")
	co.access.Lock()
	co.closed = false
	co.access.Unlock()
	return nil
}

// Close implements common.Closable.
func (co *Outbound) Close() error {
	fmt.Println("in app-commander-outbound.go func (co *Outbound) Close()")
	co.access.Lock()
	defer co.access.Unlock()

	co.closed = true
	return co.listener.Close()
}
