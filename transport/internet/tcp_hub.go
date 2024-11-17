package internet

import (
	"context"
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common/net"
)

var (
	transportListenerCache = make(map[string]ListenFunc)
)

func RegisterTransportListener(protocol string, listener ListenFunc) error {
	fmt.Println("in ttansport-internet-tcp_hub.go func RegisterTransportListener")
	if _, found := transportListenerCache[protocol]; found {
		return errors.New(" listener already registered")
	}
	transportListenerCache[protocol] = listener
	return nil
}

type ConnHandler func(Connection)

type ListenFunc func(ctx context.Context, address net.Address, port net.Port, settings *MemoryStreamConfig, handler ConnHandler) (Listener, error)

type Listener interface {
	Close() error
	Addr() net.Addr
}

// ListenUnix is the UDS version of ListenTCP
func ListenUnix(ctx context.Context, address net.Address, settings *MemoryStreamConfig, handler ConnHandler) (Listener, error) {
	fmt.Println("in ttansport-internet-tcp_hub.go func ListenUnix")
	if settings == nil {
		s, err := ToMemoryStreamConfig(nil)
		if err != nil {
			return nil, errors.New("failed to create default unix stream settings")
		}
		settings = s
	}

	protocol := settings.ProtocolName
	listenFunc := transportListenerCache[protocol]
	if listenFunc == nil {
		return nil, errors.New(" unix istener not registered")
	}
	listener, err := listenFunc(ctx, address, net.Port(0), settings, handler)
	if err != nil {
		return nil, errors.New("failed to listen on unix address")
	}
	return listener, nil
}
func ListenTCP(ctx context.Context, address net.Address, port net.Port, settings *MemoryStreamConfig, handler ConnHandler) (Listener, error) {
	fmt.Println("in ttansport-internet-tcp_hub.go func ListenTCP")
	if settings == nil {
		s, err := ToMemoryStreamConfig(nil)
		if err != nil {
			return nil, errors.New("failed to create default stream settings")
		}
		settings = s
	}
	fmt.Println("in ttansport-internet-tcp_hub.go func ListenTCP settings != nil")
	if address.Family().IsDomain() && address.Domain() == "localhost" {
		fmt.Println("in ttansport-internet-tcp_hub.go func ListenTCP address = net.LocalHostIP")
		address = net.LocalHostIP
	}

	if address.Family().IsDomain() {
		return nil, errors.New("domain address is not allowed for listening: ")
	}

	protocol := settings.ProtocolName
	listenFunc := transportListenerCache[protocol]
	if listenFunc == nil {
		return nil, errors.New(" listener not registered")
	}
	listener, err := listenFunc(ctx, address, port, settings, handler)
	if err != nil {
		return nil, errors.New("failed to listen on address")
	}
	fmt.Println("in ttansport-internet-tcp_hub.go return ListenTCP")
	return listener, nil
}

// ListenSystem listens on a local address for incoming TCP connections.
//
// v2ray:api:beta
func ListenSystem(ctx context.Context, addr net.Addr, sockopt *SocketConfig) (net.Listener, error) {
	fmt.Println("in ttansport-internet-tcp_hub.go func ListenSystem")
	return effectiveListener.Listen(ctx, addr, sockopt)
}

// ListenSystemPacket listens on a local address for incoming UDP connections.
//
// v2ray:api:beta
func ListenSystemPacket(ctx context.Context, addr net.Addr, sockopt *SocketConfig) (net.PacketConn, error) {
	fmt.Println("in ttansport-internet-tcp_hub.go func ListenSystemPacket")
	return effectiveListener.ListenPacket(ctx, addr, sockopt)
}
