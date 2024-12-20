package conf

import (
	"encoding/json"
	"sort"
	"strings"
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/app/dns"
	"github.com/gzjjjfree/gzv2ray-v4/app/router"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
)

type NameServerConfig struct {
	Address      *Address
	ClientIP     *Address
	Port         uint16
	SkipFallback bool
	Domains      []string
	ExpectIPs    StringList
}

func (c *NameServerConfig) UnmarshalJSON(data []byte) error {
	fmt.Println("in infa-conf-dns.go func (c *NameServerConfig) UnmarshalJSON")
	var address Address
	if err := json.Unmarshal(data, &address); err == nil {
		c.Address = &address
		return nil
	}

	var advanced struct {
		Address      *Address   `json:"address"`
		ClientIP     *Address   `json:"clientIp"`
		Port         uint16     `json:"port"`
		SkipFallback bool       `json:"skipFallback"`
		Domains      []string   `json:"domains"`
		ExpectIPs    StringList `json:"expectIps"`
	}
	if err := json.Unmarshal(data, &advanced); err == nil {
		c.Address = advanced.Address
		c.ClientIP = advanced.ClientIP
		c.Port = advanced.Port
		c.SkipFallback = advanced.SkipFallback
		c.Domains = advanced.Domains
		c.ExpectIPs = advanced.ExpectIPs
		return nil
	}

	return errors.New("failed to parse name server")
}

func toDomainMatchingType(t router.Domain_Type) dns.DomainMatchingType {
	fmt.Println("in infa-conf-dns.go func toDomainMatchingType")
	switch t {
	case router.Domain_Domain:
		return dns.DomainMatchingType_Subdomain
	case router.Domain_Full:
		return dns.DomainMatchingType_Full
	case router.Domain_Plain:
		return dns.DomainMatchingType_Keyword
	case router.Domain_Regex:
		return dns.DomainMatchingType_Regex
	default:
		panic("unknown domain type")
	}
}

func (c *NameServerConfig) Build() (*dns.NameServer, error) {
	fmt.Println("in infa-conf-dns.go func (c *NameServerConfig) Build() c.Address: ", c.Address)
	if c.Address == nil {
		fmt.Println("in infa-conf-dns.go func (c *NameServerConfig) Build() c.Address == nil")
		return nil, errors.New("nameServer address is not specified")
	}

	var domains []*dns.NameServer_PriorityDomain
	var originalRules []*dns.NameServer_OriginalRule

	for _, rule := range c.Domains {
		parsedDomain, err := parseDomainRule(rule)
		if err != nil {
			return nil, errors.New("invalid domain rule")
		}

		for _, pd := range parsedDomain {
			domains = append(domains, &dns.NameServer_PriorityDomain{
				Type:   toDomainMatchingType(pd.Type),
				Domain: pd.Value,
			})
		}
		originalRules = append(originalRules, &dns.NameServer_OriginalRule{
			Rule: rule,
			Size: uint32(len(parsedDomain)),
		})
	}

	geoipList, err := toCidrList(c.ExpectIPs)
	if err != nil {
		return nil, errors.New("invalid IP rule")
	}

	var myClientIP []byte
	if c.ClientIP != nil {
		fmt.Println("in infa-conf-dns.go func (c *NameServerConfig) Build() c.ClientIP != nil")
		if !c.ClientIP.Family().IsIP() {
			return nil, errors.New("not an IP address")
		}
		myClientIP = []byte(c.ClientIP.IP())
	}

	return &dns.NameServer{
		Address: &net.Endpoint{
			Network: net.Network_UDP,
			Address: c.Address.Build(),
			Port:    uint32(c.Port),
		},
		ClientIp:          myClientIP,
		SkipFallback:      c.SkipFallback,
		PrioritizedDomain: domains,
		Geoip:             geoipList,
		OriginalRules:     originalRules,
	}, nil
}

var typeMap = map[router.Domain_Type]dns.DomainMatchingType{
	router.Domain_Full:   dns.DomainMatchingType_Full,
	router.Domain_Domain: dns.DomainMatchingType_Subdomain,
	router.Domain_Plain:  dns.DomainMatchingType_Keyword,
	router.Domain_Regex:  dns.DomainMatchingType_Regex,
}

// DNSConfig is a JSON serializable object for dns.Config.
type DNSConfig struct {
	Servers         []*NameServerConfig `json:"servers"`
	Hosts           map[string]*Address `json:"hosts"`
	ClientIP        *Address            `json:"clientIp"`
	Tag             string              `json:"tag"`
	QueryStrategy   string              `json:"queryStrategy"`
	DisableCache    bool                `json:"disableCache"`
	DisableFallback bool                `json:"disableFallback"`
}

func getHostMapping(addr *Address) *dns.Config_HostMapping {
	fmt.Println("in infa-conf-dns.go func getHostMapping")
	if addr.Family().IsIP() {
		return &dns.Config_HostMapping{
			Ip: [][]byte{[]byte(addr.IP())},
		}
	}
	return &dns.Config_HostMapping{
		ProxiedDomain: addr.Domain(),
	}
}

// Build implements Buildable
func (c *DNSConfig) Build() (*dns.Config, error) {
	fmt.Println("in infa-conf-dns.go func (c *DNSConfig) Build()")
	config := &dns.Config{
		Tag:             c.Tag,
		DisableCache:    c.DisableCache,
		DisableFallback: c.DisableFallback,
	}

	if c.ClientIP != nil {
		if !c.ClientIP.Family().IsIP() {
			return nil, errors.New("not an IP address")
		}
		config.ClientIp = []byte(c.ClientIP.IP())
	}

	config.QueryStrategy = dns.QueryStrategy_USE_IP
	switch strings.ToLower(c.QueryStrategy) {
	case "useip", "use_ip", "use-ip":
		config.QueryStrategy = dns.QueryStrategy_USE_IP
	case "useip4", "useipv4", "use_ip4", "use_ipv4", "use_ip_v4", "use-ip4", "use-ipv4", "use-ip-v4":
		config.QueryStrategy = dns.QueryStrategy_USE_IP4
	case "useip6", "useipv6", "use_ip6", "use_ipv6", "use_ip_v6", "use-ip6", "use-ipv6", "use-ip-v6":
		config.QueryStrategy = dns.QueryStrategy_USE_IP6
	}

	for _, server := range c.Servers {
		ns, err := server.Build()
		if err != nil {
			return nil, errors.New("failed to build nameserver")
		}
		config.NameServer = append(config.NameServer, ns)
	}

	if c.Hosts != nil && len(c.Hosts) > 0 {
		domains := make([]string, 0, len(c.Hosts))
		for domain := range c.Hosts {
			domains = append(domains, domain)
		}
		sort.Strings(domains)

		for _, domain := range domains {
			addr := c.Hosts[domain]
			var mappings []*dns.Config_HostMapping
			switch {
			case strings.HasPrefix(domain, "domain:"):
				domainName := domain[7:]
				if len(domainName) == 0 {
					return nil, errors.New("empty domain type of rule")
				}
				mapping := getHostMapping(addr)
				mapping.Type = dns.DomainMatchingType_Subdomain
				mapping.Domain = domainName
				mappings = append(mappings, mapping)

			case strings.HasPrefix(domain, "geosite:"):
				listName := domain[8:]
				if len(listName) == 0 {
					return nil, errors.New("empty geosite rule")
				}
				domains, err := loadGeosite(listName)
				if err != nil {
					return nil, errors.New("failed to load geosite")
				}
				for _, d := range domains {
					mapping := getHostMapping(addr)
					mapping.Type = typeMap[d.Type]
					mapping.Domain = d.Value
					mappings = append(mappings, mapping)
				}

			case strings.HasPrefix(domain, "regexp:"):
				regexpVal := domain[7:]
				if len(regexpVal) == 0 {
					return nil, errors.New("empty regexp type of rule")
				}
				mapping := getHostMapping(addr)
				mapping.Type = dns.DomainMatchingType_Regex
				mapping.Domain = regexpVal
				mappings = append(mappings, mapping)

			case strings.HasPrefix(domain, "keyword:"):
				keywordVal := domain[8:]
				if len(keywordVal) == 0 {
					return nil, errors.New("empty keyword type of rule")
				}
				mapping := getHostMapping(addr)
				mapping.Type = dns.DomainMatchingType_Keyword
				mapping.Domain = keywordVal
				mappings = append(mappings, mapping)

			case strings.HasPrefix(domain, "full:"):
				fullVal := domain[5:]
				if len(fullVal) == 0 {
					return nil, errors.New("empty full domain type of rule")
				}
				mapping := getHostMapping(addr)
				mapping.Type = dns.DomainMatchingType_Full
				mapping.Domain = fullVal
				mappings = append(mappings, mapping)

			case strings.HasPrefix(domain, "dotless:"):
				mapping := getHostMapping(addr)
				mapping.Type = dns.DomainMatchingType_Regex
				switch substr := domain[8:]; {
				case substr == "":
					mapping.Domain = "^[^.]*$"
				case !strings.Contains(substr, "."):
					mapping.Domain = "^[^.]*" + substr + "[^.]*$"
				default:
					return nil, errors.New("substr in dotless rule should not contain a dot")
				}
				mappings = append(mappings, mapping)

			case strings.HasPrefix(domain, "ext:"):
				kv := strings.Split(domain[4:], ":")
				if len(kv) != 2 {
					return nil, errors.New("invalid external resource")
				}
				filename := kv[0]
				list := kv[1]
				domains, err := loadGeositeWithAttr(filename, list)
				if err != nil {
					return nil, errors.New("failed to load domain list")
				}
				for _, d := range domains {
					mapping := getHostMapping(addr)
					mapping.Type = typeMap[d.Type]
					mapping.Domain = d.Value
					mappings = append(mappings, mapping)
				}

			default:
				mapping := getHostMapping(addr)
				mapping.Type = dns.DomainMatchingType_Full
				mapping.Domain = domain
				mappings = append(mappings, mapping)
			}

			config.StaticHosts = append(config.StaticHosts, mappings...)
		}
	}

	return config, nil
}
