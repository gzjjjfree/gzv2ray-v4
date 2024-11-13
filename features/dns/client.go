package dns

import (
	"errors"


	"github.com/gzjjjfree/gzv2ray-v4/features"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/serial"
)

// ClientType returns the type of Client interface. Can be used for implementing common.HasType.
// ClientType 返回 Client 接口的类型。可用于实现 common.HasType。
// v2ray:api:beta
func ClientType() interface{} {
	return (*Client)(nil)
}

// Client is a V2Ray feature for querying DNS information.
// 客户端是 V2Ray 的一个查询 DNS 信息的功能
// v2ray:api:stable
type Client interface {
	features.Feature

	// LookupIP returns IP address for the given domain. IPs may contain IPv4 and/or IPv6 addresses.
	// LookupIP 返回给定域的 IP 地址。IP 可能包含 IPv4 和/或 IPv6 地址。
	LookupIP(domain string) ([]net.IP, error)
}


// IPOption is an object for IP query options.
type IPOption struct {
	IPv4Enable bool
	IPv6Enable bool
	FakeEnable bool
}

// IPv4Lookup is an optional feature for querying IPv4 addresses only.
//
// v2ray:api:beta
type IPv4Lookup interface {
	LookupIPv4(domain string) ([]net.IP, error)
}

// IPv6Lookup is an optional feature for querying IPv6 addresses only.
//
// v2ray:api:beta
type IPv6Lookup interface {
	LookupIPv6(domain string) ([]net.IP, error)
}

// ClientWithIPOption is an optional feature for querying DNS information.
//
// v2ray:api:beta
type ClientWithIPOption interface {
	// GetIPOption returns IPOption for the DNS client.
	GetIPOption() *IPOption

	// SetQueryOption sets IPv4Enable and IPv6Enable for the DNS client.
	SetQueryOption(isIPv4Enable, isIPv6Enable bool)

	// SetFakeDNSOption sets FakeEnable option for DNS client.
	SetFakeDNSOption(isFakeEnable bool)
}

// ErrEmptyResponse indicates that DNS query succeeded but no answer was returned.
var ErrEmptyResponse = errors.New("empty response")

type RCodeError uint16

func (e RCodeError) Error() string {
	return serial.Concat("rcode: ", uint16(e))
}

func RCodeFromError(err error) uint16 {
	if err == nil {
		return 0
	}
	cause := errors.Unwrap(err)
	if r, ok := cause.(RCodeError); ok {
		return uint16(r)
	}
	return 0
}
