package net

import (
	"net"
	"strings"
	"fmt"
)

// Destination represents a network destination including address and protocol (tcp / udp).
type Destination struct {
	Address Address
	Port    Port
	Network Network
}

// DestinationFromAddr generates a Destination from a net address.
// DestinationFromAddr 从网络地址生成目标。
func DestinationFromAddr(addr net.Addr) Destination {
fmt.Println("in common-net-destination.go func DestinationFromAddr")
	switch addr := addr.(type) {
	case *net.TCPAddr:
		return TCPDestination(IPAddress(addr.IP), Port(addr.Port))
	case *net.UDPAddr:
		return UDPDestination(IPAddress(addr.IP), Port(addr.Port))
	case *net.UnixAddr:
		return UnixDestination(DomainAddress(addr.Name))
	default:
		panic("Net: Unknown address type.")
	}
}

// ParseDestination converts a destination from its string presentation.
func ParseDestination(dest string) (Destination, error) {
	fmt.Println("in common-net-destination.go func  ParseDestination")
	d := Destination{
		Address: AnyIP,
		Port:    Port(0),
	}

	switch {
	case strings.HasPrefix(dest, "tcp:"):
		d.Network = Network_TCP
		dest = dest[4:]
	case strings.HasPrefix(dest, "udp:"):
		d.Network = Network_UDP
		dest = dest[4:]
	case strings.HasPrefix(dest, "unix:"):
		d = UnixDestination(DomainAddress(dest[5:]))
		return d, nil
	}

	hstr, pstr, err := SplitHostPort(dest)
	if err != nil {
		return d, err
	}
	if len(hstr) > 0 {
		d.Address = ParseAddress(hstr)
	}
	if len(pstr) > 0 {
		port, err := PortFromString(pstr)
		if err != nil {
			return d, err
		}
		d.Port = port
	}
	return d, nil
}

// TCPDestination creates a TCP destination with given address
func TCPDestination(address Address, port Port) Destination {
	return Destination{
		Network: Network_TCP,
		Address: address,
		Port:    port,
	}
}

// UDPDestination creates a UDP destination with given address
func UDPDestination(address Address, port Port) Destination {
	return Destination{
		Network: Network_UDP,
		Address: address,
		Port:    port,
	}
}

// UnixDestination creates a Unix destination with given address
func UnixDestination(address Address) Destination {
	return Destination{
		Network: Network_UNIX,
		Address: address,
	}
}

// NetAddr returns the network address in this Destination in string form.
func (d Destination) NetAddr() string {
	addr := ""
	if d.Network == Network_TCP || d.Network == Network_UDP {
		addr = d.Address.String() + ":" + d.Port.String()
	} else if d.Network == Network_UNIX {
		addr = d.Address.String()
	}
	return addr
}

// String returns the strings form of this Destination.
func (d Destination) String() string {
	prefix := "unknown:"
	switch d.Network {
	case Network_TCP:
		prefix = "tcp:"
	case Network_UDP:
		prefix = "udp:"
	case Network_UNIX:
		prefix = "unix:"
	}
	return prefix + d.NetAddr()
}

// IsValid returns true if this Destination is valid.
// 如果此目的地有效，则 IsValid 返回 true。
func (d Destination) IsValid() bool {
	return d.Network != Network_Unknown
}

// AsDestination converts current Endpoint into Destination.
func (p *Endpoint) AsDestination() Destination {
	return Destination{
		Network: p.Network,
		Address: p.Address.AsAddress(),
		Port:    Port(p.Port),
	}
}
