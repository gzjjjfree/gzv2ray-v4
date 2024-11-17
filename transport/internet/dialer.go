package internet

import (
	"context"
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/session"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/tagged"
)

// Dialer is the interface for dialing outbound connections.
type Dialer interface {
	// Dial dials a system connection to the given destination.
	Dial(ctx context.Context, destination net.Destination) (Connection, error)

	// Address returns the address used by this Dialer. Maybe nil if not known.
	Address() net.Address
}

// dialFunc is an interface to dial network connection to a specific destination.
type dialFunc func(ctx context.Context, dest net.Destination, streamSettings *MemoryStreamConfig) (Connection, error)

var (
	transportDialerCache = make(map[string]dialFunc)
)

// RegisterTransportDialer registers a Dialer with given name.
func RegisterTransportDialer(protocol string, dialer dialFunc) error {
	fmt.Println("in transport-internet-dialer.go func RegisterTransportDialer protocol is: ", protocol)
	if _, found := transportDialerCache[protocol]; found {
		return errors.New(" dialer already registered")
	}
	transportDialerCache[protocol] = dialer
	return nil
}

// Dial dials a internet connection towards the given destination.
func Dial(ctx context.Context, dest net.Destination, streamSettings *MemoryStreamConfig) (Connection, error) {
	fmt.Println("in transport-internet-dialer.go func Dial dest is: ", dest, " dest.Network is: ", dest.Network)
	if dest.Network == net.Network_TCP {
		if streamSettings == nil {
			fmt.Println("in transport-internet-dialer.go func Dial  streamSettings == nil")
			s, err := ToMemoryStreamConfig(nil)
			if err != nil {
				return nil, errors.New("failed to create default stream settings")
			}
			streamSettings = s
		}

		protocol := streamSettings.ProtocolName
		dialer := transportDialerCache[protocol]
		if dialer == nil {
			return nil, errors.New(" dialer not registered")
		}
		return dialer(ctx, dest, streamSettings)
	}

	if dest.Network == net.Network_UDP {
		udpDialer := transportDialerCache["udp"]
		if udpDialer == nil {
			return nil, errors.New("UDP dialer not registered")
		}
		return udpDialer(ctx, dest, streamSettings)
	}

	return nil, errors.New("unknown network ")
}

// DialSystem calls system dialer to create a network connection.
func DialSystem(ctx context.Context, dest net.Destination, sockopt *SocketConfig) (net.Conn, error) {
	fmt.Println("in transport-internet-dialer.go func DialSystem")
	var src net.Address
	if outbound := session.OutboundFromContext(ctx); outbound != nil {
		src = outbound.Gateway
	}

	if transportLayerOutgoingTag := session.GetTransportLayerProxyTagFromContext(ctx); transportLayerOutgoingTag != "" {
		return DialTaggedOutbound(ctx, dest, transportLayerOutgoingTag)
	}

	return effectiveSystemDialer.Dial(ctx, src, dest, sockopt)
}

func DialTaggedOutbound(ctx context.Context, dest net.Destination, tag string) (net.Conn, error) {
	fmt.Println("in transport-internet-dialer.go func DialTaggedOutbound")
	if tagged.Dialer == nil {
		return nil, errors.New("tagged dial not enabled")
	}
	return tagged.Dialer(ctx, dest, tag)
}
