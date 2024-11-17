// +build !confonly

package dns

import (
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/strmatcher"
	"github.com/gzjjjfree/gzv2ray-v4/common/uuid"
)

var typeMap = map[DomainMatchingType]strmatcher.Type{
	DomainMatchingType_Full:      strmatcher.Full,
	DomainMatchingType_Subdomain: strmatcher.Domain,
	DomainMatchingType_Keyword:   strmatcher.Substr,
	DomainMatchingType_Regex:     strmatcher.Regex,
}

// References:
// https://www.iana.org/assignments/special-use-domain-names/special-use-domain-names.xhtml
// https://unix.stackexchange.com/questions/92441/whats-the-difference-between-local-home-and-lan
var localTLDsAndDotlessDomains = []*NameServer_PriorityDomain{
	{Type: DomainMatchingType_Regex, Domain: "^[^.]+$"}, // This will only match domains without any dot
	{Type: DomainMatchingType_Subdomain, Domain: "local"},
	{Type: DomainMatchingType_Subdomain, Domain: "localdomain"},
	{Type: DomainMatchingType_Subdomain, Domain: "localhost"},
	{Type: DomainMatchingType_Subdomain, Domain: "lan"},
	{Type: DomainMatchingType_Subdomain, Domain: "home.arpa"},
	{Type: DomainMatchingType_Subdomain, Domain: "example"},
	{Type: DomainMatchingType_Subdomain, Domain: "invalid"},
	{Type: DomainMatchingType_Subdomain, Domain: "test"},
}

var localTLDsAndDotlessDomainsRule = &NameServer_OriginalRule{
	Rule: "geosite:private",
	Size: uint32(len(localTLDsAndDotlessDomains)),
}

func toStrMatcher(t DomainMatchingType, domain string) (strmatcher.Matcher, error) {
	fmt.Println("in app-dns-config.go func toStrMatcher")
	strMType, f := typeMap[t]
	if !f {
		return nil, errors.New("unknown mapping type")
	}
	matcher, err := strMType.New(domain)
	if err != nil {
		return nil, errors.New("failed to create str matcher")
	}
	return matcher, nil
}

func toNetIP(addrs []net.Address) ([]net.IP, error) {
	fmt.Println("in app-dns-config.go func toNetIP")
	ips := make([]net.IP, 0, len(addrs))
	for _, addr := range addrs {
		if addr.Family().IsIP() {
			ips = append(ips, addr.IP())
		} else {
			return nil, errors.New("failed to convert address to Net IP")
		}
	}
	return ips, nil
}

func generateRandomTag() string {
	id := uuid.New()
	return "gzv2ray.system." + id.String()
}
