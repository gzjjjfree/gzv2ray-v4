// +build !confonly

package dns

import (
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/strmatcher"
	"github.com/gzjjjfree/gzv2ray-v4/features"
	"github.com/gzjjjfree/gzv2ray-v4/features/dns"
)

// StaticHosts represents static domain-ip mapping in DNS server.
type StaticHosts struct {
	ips      [][]net.Address
	matchers *strmatcher.MatcherGroup
}

// NewStaticHosts creates a new StaticHosts instance.
func NewStaticHosts(hosts []*Config_HostMapping, legacy map[string]*net.IPOrDomain) (*StaticHosts, error) {
	fmt.Println("in app-dns-hosts.go func NewStaticHosts")
	g := new(strmatcher.MatcherGroup)
	sh := &StaticHosts{
		ips:      make([][]net.Address, len(hosts)+len(legacy)+16),
		matchers: g,
	}

	if legacy != nil {
		features.PrintDeprecatedFeatureWarning("simple host mapping")

		for domain, ip := range legacy {
			matcher, err := strmatcher.Full.New(domain)
			common.Must(err)
			id := g.Add(matcher)

			address := ip.AsAddress()
			if address.Family().IsDomain() {
				return nil, errors.New("invalid domain address in static hosts")
			}

			sh.ips[id] = []net.Address{address}
		}
	}

	for _, mapping := range hosts {
		matcher, err := toStrMatcher(mapping.Type, mapping.Domain)
		if err != nil {
			return nil, errors.New("failed to create domain matcher")
		}
		id := g.Add(matcher)
		ips := make([]net.Address, 0, len(mapping.Ip)+1)
		switch {
		case len(mapping.Ip) > 0:
			for _, ip := range mapping.Ip {
				addr := net.IPAddress(ip)
				if addr == nil {
					return nil, errors.New("invalid IP address in static hosts")
				}
				ips = append(ips, addr)
			}

		case len(mapping.ProxiedDomain) > 0:
			ips = append(ips, net.DomainAddress(mapping.ProxiedDomain))

		default:
			return nil, errors.New("neither IP address nor proxied domain specified for domain")
		}

		// Special handling for localhost IPv6. This is a dirty workaround as JSON config supports only single IP mapping.
		if len(ips) == 1 && ips[0] == net.LocalHostIP {
			ips = append(ips, net.LocalHostIPv6)
		}

		sh.ips[id] = ips
	}

	return sh, nil
}

func filterIP(ips []net.Address, option dns.IPOption) []net.Address {
	fmt.Println("in app-dns-hosts.go func filterIP")
	filtered := make([]net.Address, 0, len(ips))
	for _, ip := range ips {
		if (ip.Family().IsIPv4() && option.IPv4Enable) || (ip.Family().IsIPv6() && option.IPv6Enable) {
			filtered = append(filtered, ip)
		}
	}
	return filtered
}

func (h *StaticHosts) lookupInternal(domain string) []net.Address {
	var ips []net.Address
	for _, id := range h.matchers.Match(domain) {
		ips = append(ips, h.ips[id]...)
	}
	return ips
}

func (h *StaticHosts) lookup(domain string, option dns.IPOption, maxDepth int) []net.Address {
	fmt.Println("in app-dns-hosts.go func (h *StaticHosts) lookup")
	switch addrs := h.lookupInternal(domain); {
	case len(addrs) == 0: // Not recorded in static hosts, return nil
		return nil
	case len(addrs) == 1 && addrs[0].Family().IsDomain(): // Try to unwrap domain
		if maxDepth > 0 {
			unwrapped := h.lookup(addrs[0].Domain(), option, maxDepth-1)
			if unwrapped != nil {
				return unwrapped
			}
		}
		return addrs
	default: // IP record found, return a non-nil IP array
		return filterIP(addrs, option)
	}
}

// Lookup returns IP addresses or proxied domain for the given domain, if exists in this StaticHosts.
func (h *StaticHosts) Lookup(domain string, option dns.IPOption) []net.Address {
	return h.lookup(domain, option, 5)
}
