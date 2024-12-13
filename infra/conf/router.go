package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/gzjjjfree/gzv2ray-v4/app/router"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/platform/filesystem"
)

type RouterRulesConfig struct {
	RuleList       []json.RawMessage `json:"rules"`
	DomainStrategy string            `json:"domainStrategy"`
}

type BalancingRule struct {
	Tag       string     `json:"tag"`
	Selectors StringList `json:"selector"`
}

func (r *BalancingRule) Build() (*router.BalancingRule, error) {
	fmt.Println("in infra-conf-router.go func (r *BalancingRule) Build()")
	if r.Tag == "" {
		return nil, errors.New("empty balancer tag")
	}
	if len(r.Selectors) == 0 {
		return nil, errors.New("empty selector list")
	}

	return &router.BalancingRule{
		Tag:              r.Tag,
		OutboundSelector: []string(r.Selectors),
	}, nil
}

type RouterConfig struct {
	Settings       *RouterRulesConfig `json:"settings"` // Deprecated
	RuleList       []json.RawMessage  `json:"rules"`
	DomainStrategy *string            `json:"domainStrategy"`
	Balancers      []*BalancingRule   `json:"balancers"`

	DomainMatcher string `json:"domainMatcher"`
}

func (c *RouterConfig) getDomainStrategy() router.Config_DomainStrategy {
	fmt.Println("in infra-conf-router.go func (c *RouterConfig) getDomainStrategy()")
	ds := ""
	if c.DomainStrategy != nil {
		ds = *c.DomainStrategy
	} else if c.Settings != nil {
		ds = c.Settings.DomainStrategy
	}

	switch strings.ToLower(ds) {
	case "alwaysip", "always_ip", "always-ip":
		return router.Config_UseIp
	case "ipifnonmatch", "ip_if_non_match", "ip-if-non-match":
		return router.Config_IpIfNonMatch
	case "ipondemand", "ip_on_demand", "ip-on-demand":
		return router.Config_IpOnDemand
	default:
		return router.Config_AsIs
	}
}

func (c *RouterConfig) Build() (*router.Config, error) {
	fmt.Println("in infra-conf-router.go func  (c *RouterConfig) Build()")
	config := new(router.Config)
	// 取得路由模式
	config.DomainStrategy = c.getDomainStrategy()

	var rawRuleList []json.RawMessage
	if c != nil {
		rawRuleList = c.RuleList
		if c.Settings != nil {
			fmt.Println("in infra-conf-router.go func  (c *RouterConfig) Build() c.Settings != nil")
			// config.jjson 的 routing-settings 字段不为空时，将值加入路由列表
			c.RuleList = append(c.RuleList, c.Settings.RuleList...)
			rawRuleList = c.RuleList
		}
	}

	for _, rawRule := range rawRuleList {
		// 解析每一个路由, rule = paresfieldRule 结构体
		rule, err := ParseRule(rawRule)
		if err != nil {
			return nil, err
		}

		if rule.DomainMatcher == "" {
			rule.DomainMatcher = c.DomainMatcher
		}

		//汇总每个路由
		config.Rule = append(config.Rule, rule)
	}
	// balancers 字段是一个数组，每个元素代表一种负载均衡算法
	for _, rawBalancer := range c.Balancers {
		balancer, err := rawBalancer.Build()
		if err != nil {
			return nil, err
		}
		config.BalancingRule = append(config.BalancingRule, balancer)
	}
	return config, nil
}

type RouterRule struct {
	Type        string `json:"type"`
	OutboundTag string `json:"outboundTag"`
	BalancerTag string `json:"balancerTag"`

	DomainMatcher string `json:"domainMatcher"`
}

func ParseIP(s string) (*router.CIDR, error) {
	fmt.Println("in infra-conf-router.go func ParseIP")
	var addr, mask string
	i := strings.Index(s, "/")
	if i < 0 {
		addr = s
	} else {
		addr = s[:i]
		mask = s[i+1:]
	}
	ip := net.ParseAddress(addr)
	switch ip.Family() {
	case net.AddressFamilyIPv4:
		bits := uint32(32)
		if len(mask) > 0 {
			bits64, err := strconv.ParseUint(mask, 10, 32)
			if err != nil {
				return nil, errors.New("invalid network mask for router")
			}
			bits = uint32(bits64)
		}
		if bits > 32 {
			return nil, errors.New("invalid network mask for router")
		}
		return &router.CIDR{
			Ip:     []byte(ip.IP()),
			Prefix: bits,
		}, nil
	case net.AddressFamilyIPv6:
		bits := uint32(128)
		if len(mask) > 0 {
			bits64, err := strconv.ParseUint(mask, 10, 32)
			if err != nil {
				return nil, errors.New("invalid network mask for router")
			}
			bits = uint32(bits64)
		}
		if bits > 128 {
			return nil, errors.New("invalid network mask for router")
		}
		return &router.CIDR{
			Ip:     []byte(ip.IP()),
			Prefix: bits,
		}, nil
	default:
		return nil, errors.New("unsupported address for router")
	}
}

func loadGeoIP(country string) ([]*router.CIDR, error) {
	fmt.Println("in infra-conf-router.go func loadGeoIP")
	return loadIP("geoip.dat", country)
}

func loadIP(filename, country string) ([]*router.CIDR, error) {
	fmt.Println("in infra-conf-router.go func loadIP")
	geoipBytes, err := filesystem.ReadAsset(filename)
	if err != nil {
		return nil, errors.New("failed to open fil")
	}
	var geoipList router.GeoIPList
	if err := proto.Unmarshal(geoipBytes, &geoipList); err != nil {
		return nil, err
	}

	for _, geoip := range geoipList.Entry {
		if strings.EqualFold(geoip.CountryCode, country) {
			return geoip.Cidr, nil
		}
	}

	return nil, errors.New("country not found in")
}

func loadSite(filename, list string) ([]*router.Domain, error) {
	fmt.Println("in infra-conf-router.go func loadSite: ", filename)
	// 将文件读入 geositeBytes
	geositeBytes, err := filesystem.ReadAsset(filename)
	if err != nil {
		fmt.Println("in infra-conf-router.go func loadSite err != nil")
		return nil, errors.New("failed to open file")
	}
	var geositeList router.GeoSiteList
	// 将字节反序列化为结构体 geositeList
	if err := proto.Unmarshal(geositeBytes, &geositeList); err != nil {
		fmt.Println("in infra-conf-router.go func loadSite proto.Unmarshal err")
		return nil, err
	}
	//fmt.Println("in infra-conf-router.go func loadSite range geositeList.Entry")
	for _, site := range geositeList.Entry {
		// 根据条目返回Domain
		if strings.EqualFold(site.CountryCode, list) {
			//fmt.Println("in infra-conf-router.go func loadSite: ", filename)
			//return nil, errors.New("list not found in")
			return site.Domain, nil
		}
	}

	return nil, errors.New("list not found in")
}

type AttributeMatcher interface {
	Match(*router.Domain) bool
}

type BooleanMatcher string

func (m BooleanMatcher) Match(domain *router.Domain) bool {
	fmt.Println("in infra-conf-router.go func  (m BooleanMatcher) Match")
	for _, attr := range domain.Attribute {
		if strings.EqualFold(attr.GetKey(), string(m)) {
			return true
		}
	}
	return false
}

type AttributeList struct {
	matcher []AttributeMatcher
}

func (al *AttributeList) Match(domain *router.Domain) bool {
	fmt.Println("in infra-conf-router.go func (al *AttributeList) Match")
	for _, matcher := range al.matcher {
		if !matcher.Match(domain) {
			return false
		}
	}
	return true
}

func (al *AttributeList) IsEmpty() bool {
	return len(al.matcher) == 0
}

func parseAttrs(attrs []string) *AttributeList {
	fmt.Println("in infra-conf-router.go func parseAttrs")
	al := new(AttributeList)
	for _, attr := range attrs {
		trimmedAttr := strings.ToLower(strings.TrimSpace(attr))
		if len(trimmedAttr) == 0 {
			continue
		}
		al.matcher = append(al.matcher, BooleanMatcher(trimmedAttr))
	}
	return al
}

func loadGeosite(list string) ([]*router.Domain, error) {
	return loadGeositeWithAttr("geosite.dat", list)
}

func loadGeositeWithAttr(file string, siteWithAttr string) ([]*router.Domain, error) {
	fmt.Println("in infra-conf-router.go func loadGeositeWithAttr")
	// 将一个字符串 siteWithAttr 根据 @ 符号进行分割
	parts := strings.Split(siteWithAttr, "@")
	if len(parts) == 0 {
		return nil, errors.New("empty rule")
	}
	fmt.Println("in infra-conf-router.go func loadGeositeWithAttr parts: ", parts)
	// TrimSpace 返回字符串 s 的切片，其中所有前导和尾随空格均被删除，如 Unicode 所定义。
	list := strings.TrimSpace(parts[0])
	attrVal := parts[1:]

	if len(list) == 0 {
		return nil, errors.New("empty listname in rule")
	}

	// 读取文件，赋值给 domains
	domains, err := loadSite(file, list)
	if err != nil {
		return nil, err
	}

	attrs := parseAttrs(attrVal)
	fmt.Println("in infra-conf-router.go func loadGeositeWithAttr attrs: ", attrs)
	if attrs.IsEmpty() {
		fmt.Println("in infra-conf-router.go func loadGeositeWithAttr attrs.IsEmpty() ")
		if strings.Contains(siteWithAttr, "@") {
			errors.New("empty attribute list")
		}
		return domains, nil
	}

	filteredDomains := make([]*router.Domain, 0, len(domains))
	fmt.Println("in infra-conf-router.go func loadGeositeWithAttr filteredDomains: ", filteredDomains)
	hasAttrMatched := false
	for _, domain := range domains {
		if attrs.Match(domain) {
			fmt.Println("in infra-conf-router.go func loadGeositeWithAttr attrs.Match(domain): ", attrs.Match(domain))
			hasAttrMatched = true
			filteredDomains = append(filteredDomains, domain)
		}
	}
	if !hasAttrMatched {
		errors.New("attribute match no rule: geosite")
	}

	return filteredDomains, nil
}

func parseDomainRule(domain string) ([]*router.Domain, error) {
	fmt.Println("in infra-conf-router.go func parseDomainRule")
	// HasPrefix 报告字符串 s 是否以前缀开头。
	if strings.HasPrefix(domain, "geosite:") {
		list := domain[8:]
		// 取: 后的值，空报错
		if len(list) == 0 {
			return nil, errors.New("empty listname in rule")
		}
		// domains 为读取到文件的 []*router.Domain 列表
		domains, err := loadGeosite(list)
		if err != nil {
			return nil, errors.New("failed to load geosite")
		}

		return domains, nil
	}

	var isExtDatFile = 0
	{
		const prefix = "ext:"
		if strings.HasPrefix(domain, prefix) {
			isExtDatFile = len(prefix)
		}
		const prefixQualified = "ext-domain:"
		if strings.HasPrefix(domain, prefixQualified) {
			isExtDatFile = len(prefixQualified)
		}
	}

	if isExtDatFile != 0 {
		kv := strings.Split(domain[isExtDatFile:], ":")
		if len(kv) != 2 {
			return nil, errors.New("invalid external resource")
		}
		filename := kv[0]
		list := kv[1]
		domains, err := loadGeositeWithAttr(filename, list)
		if err != nil {
			return nil, errors.New("failed to load external geosite")
		}

		return domains, nil
	}

	domainRule := new(router.Domain)
	switch {
	case strings.HasPrefix(domain, "regexp:"):
		regexpVal := domain[7:]
		if len(regexpVal) == 0 {
			return nil, errors.New("empty regexp type of rule")
		}
		domainRule.Type = router.Domain_Regex
		domainRule.Value = regexpVal

	case strings.HasPrefix(domain, "domain:"):
		domainName := domain[7:]
		if len(domainName) == 0 {
			return nil, errors.New("empty domain type of rule")
		}
		domainRule.Type = router.Domain_Domain
		domainRule.Value = domainName

	case strings.HasPrefix(domain, "full:"):
		fullVal := domain[5:]
		if len(fullVal) == 0 {
			return nil, errors.New("empty full domain type of rule")
		}
		domainRule.Type = router.Domain_Full
		domainRule.Value = fullVal

	case strings.HasPrefix(domain, "keyword:"):
		keywordVal := domain[8:]
		if len(keywordVal) == 0 {
			return nil, errors.New("empty keyword type of rule")
		}
		domainRule.Type = router.Domain_Plain
		domainRule.Value = keywordVal

	case strings.HasPrefix(domain, "dotless:"):
		domainRule.Type = router.Domain_Regex
		switch substr := domain[8:]; {
		case substr == "":
			domainRule.Value = "^[^.]*$"
		case !strings.Contains(substr, "."):
			domainRule.Value = "^[^.]*" + substr + "[^.]*$"
		default:
			return nil, errors.New("substr in dotless rule should not contain a dot")
		}

	default:
		domainRule.Type = router.Domain_Plain
		domainRule.Value = domain
	}
	return []*router.Domain{domainRule}, nil
}

func toCidrList(ips StringList) ([]*router.GeoIP, error) {
	fmt.Println("in infra-conf-router.go func toCidrList")
	var geoipList []*router.GeoIP
	var customCidrs []*router.CIDR

	for _, ip := range ips {
		if strings.HasPrefix(ip, "geoip:") {
			country := ip[6:]
			isReverseMatch := false
			if strings.HasPrefix(ip, "geoip:!") {
				country = ip[7:]
				isReverseMatch = true
			}
			if len(country) == 0 {
				return nil, errors.New("empty country name in rule")
			}
			geoip, err := loadGeoIP(country)
			if err != nil {
				return nil, errors.New("failed to load geoip")
			}

			geoipList = append(geoipList, &router.GeoIP{
				CountryCode:  strings.ToUpper(country),
				Cidr:         geoip,
				ReverseMatch: isReverseMatch,
			})

			continue
		}

		var isExtDatFile = 0
		{
			const prefix = "ext:"
			if strings.HasPrefix(ip, prefix) {
				isExtDatFile = len(prefix)
			}
			const prefixQualified = "ext-ip:"
			if strings.HasPrefix(ip, prefixQualified) {
				isExtDatFile = len(prefixQualified)
			}
		}

		if isExtDatFile != 0 {
			kv := strings.Split(ip[isExtDatFile:], ":")
			if len(kv) != 2 {
				return nil, errors.New("invalid external resource")
			}

			filename := kv[0]
			country := kv[1]
			if len(filename) == 0 || len(country) == 0 {
				return nil, errors.New("empty filename or empty country in rule")
			}

			isReverseMatch := false
			if strings.HasPrefix(country, "!") {
				country = country[1:]
				isReverseMatch = true
			}
			geoip, err := loadIP(filename, country)
			if err != nil {
				return nil, errors.New("failed to load geoip")
			}

			geoipList = append(geoipList, &router.GeoIP{
				CountryCode:  strings.ToUpper(filename + "_" + country),
				Cidr:         geoip,
				ReverseMatch: isReverseMatch,
			})

			continue
		}

		ipRule, err := ParseIP(ip)
		if err != nil {
			return nil, errors.New("invalid IP")
		}
		customCidrs = append(customCidrs, ipRule)
	}

	if len(customCidrs) > 0 {
		geoipList = append(geoipList, &router.GeoIP{
			Cidr: customCidrs,
		})
	}

	return geoipList, nil
}

func parseFieldRule(msg json.RawMessage) (*router.RoutingRule, error) {
	fmt.Println("in infra-conf-router.go func parseFieldRule")
	type RawFieldRule struct {
		RouterRule
		Domain     *StringList  `json:"domain"`
		Domains    *StringList  `json:"domains"`
		IP         *StringList  `json:"ip"`
		Port       *PortList    `json:"port"`
		Network    *NetworkList `json:"network"`
		SourceIP   *StringList  `json:"source"`
		SourcePort *PortList    `json:"sourcePort"`
		User       *StringList  `json:"user"`
		InboundTag *StringList  `json:"inboundTag"`
		Protocols  *StringList  `json:"protocol"`
		Attributes string       `json:"attrs"`
	}
	rawFieldRule := new(RawFieldRule)
	// 将单个路由设置解析为 RawFieldRule 结构体
	err := json.Unmarshal(msg, rawFieldRule)
	if err != nil {
		return nil, err
	}

	rule := new(router.RoutingRule)
	switch {
	case len(rawFieldRule.OutboundTag) > 0:
		rule.TargetTag = &router.RoutingRule_Tag{
			Tag: rawFieldRule.OutboundTag,
		}
	case len(rawFieldRule.BalancerTag) > 0:
		rule.TargetTag = &router.RoutingRule_BalancingTag{
			BalancingTag: rawFieldRule.BalancerTag,
		}
	default:
		return nil, errors.New("neither outboundTag nor balancerTag is specified in routing rule")
	}

	if rawFieldRule.DomainMatcher != "" {
		rule.DomainMatcher = rawFieldRule.DomainMatcher
	}
	//fmt.Println("in infra-conf-router.go func parseFieldRule rawFieldRule.Domain")
	if rawFieldRule.Domain != nil {
		for _, domain := range *rawFieldRule.Domain {
			rules, err := parseDomainRule(domain)
			if err != nil {
				return nil, errors.New("failed to parse domain rule")
			}
			rule.Domain = append(rule.Domain, rules...)
		}
	}
	//fmt.Println("in infra-conf-router.go func parseFieldRule rawFieldRule.Domains")
	if rawFieldRule.Domains != nil {
		for _, domain := range *rawFieldRule.Domains {
			rules, err := parseDomainRule(domain)
			if err != nil {
				return nil, errors.New("failed to parse domain rule")
			}
			rule.Domain = append(rule.Domain, rules...)
		}
	}
	//fmt.Println("in infra-conf-router.go func parseFieldRule rawFieldRule.IP")
	if rawFieldRule.IP != nil {
		geoipList, err := toCidrList(*rawFieldRule.IP)
		if err != nil {
			return nil, err
		}
		rule.Geoip = geoipList
	}

	if rawFieldRule.Port != nil {
		rule.PortList = rawFieldRule.Port.Build()
	}

	if rawFieldRule.Network != nil {
		rule.Networks = rawFieldRule.Network.Build()
	}

	if rawFieldRule.SourceIP != nil {
		geoipList, err := toCidrList(*rawFieldRule.SourceIP)
		if err != nil {
			return nil, err
		}
		rule.SourceGeoip = geoipList
	}

	if rawFieldRule.SourcePort != nil {
		rule.SourcePortList = rawFieldRule.SourcePort.Build()
	}

	if rawFieldRule.User != nil {
		for _, s := range *rawFieldRule.User {
			rule.UserEmail = append(rule.UserEmail, s)
		}
	}

	if rawFieldRule.InboundTag != nil {
		for _, s := range *rawFieldRule.InboundTag {
			rule.InboundTag = append(rule.InboundTag, s)
		}
	}

	if rawFieldRule.Protocols != nil {
		for _, s := range *rawFieldRule.Protocols {
			rule.Protocol = append(rule.Protocol, s)
		}
	}

	if len(rawFieldRule.Attributes) > 0 {
		rule.Attributes = rawFieldRule.Attributes
	}

	return rule, nil
}

func ParseRule(msg json.RawMessage) (*router.RoutingRule, error) {
	fmt.Printf("in infra-conf-router.go func ParseRule msg: %s\n", msg)
	rawRule := new(RouterRule)
	// 将每个路由字段解析为结构体 RouterrRule,这次解析只为了取得type来判断？
	err := json.Unmarshal(msg, rawRule)
	if err != nil {
		return nil, errors.New("invalid router rule")
	}
	//fmt.Printf("in infra-conf-router.go func ParseRule msg2: %s\n", msg)
	// 路由字段的 type 必须为 field
	if strings.EqualFold(rawRule.Type, "field") {
		// type 正确时，
		fieldrule, err := parseFieldRule(msg)
		if err != nil {
			return nil, errors.New("invalid field rule")
		}
		return fieldrule, nil
	}

	return nil, errors.New("unknown router rule type")
}
