// +build !confonly

package router

import (
	"strings"
	"fmt"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"

	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/strmatcher"
	"github.com/gzjjjfree/gzv2ray-v4/features/routing"
)

type Condition interface {
	Apply(ctx routing.Context) bool
}

type ConditionChan []Condition

func NewConditionChan() *ConditionChan {
	var condChan ConditionChan = make([]Condition, 0, 8)
	return &condChan
}

func (v *ConditionChan) Add(cond Condition) *ConditionChan {
	*v = append(*v, cond)
	return v
}

// Apply applies all conditions registered in this chan.
func (v *ConditionChan) Apply(ctx routing.Context) bool {
	for _, cond := range *v {
		if !cond.Apply(ctx) {
			return false
		}
	}
	return true
}

func (v *ConditionChan) Len() int {
	return len(*v)
}

var matcherTypeMap = map[Domain_Type]strmatcher.Type{
	Domain_Plain:  strmatcher.Substr,
	Domain_Regex:  strmatcher.Regex,
	Domain_Domain: strmatcher.Domain,
	Domain_Full:   strmatcher.Full,
}

func domainToMatcher(domain *Domain) (strmatcher.Matcher, error) {
	//fmt.Println("in app-router-condition.go func domainToMatcher")
	matcherType, f := matcherTypeMap[domain.Type]
	if !f {
		return nil, newError("unsupported domain type", domain.Type)
	}

	matcher, err := matcherType.New(domain.Value)
	if err != nil {
		return nil, newError("failed to create domain matcher").Base(err)
	}

	return matcher, nil
}

type DomainMatcher struct {
	matchers strmatcher.IndexMatcher
}

func NewMphMatcherGroup(domains []*Domain) (*DomainMatcher, error) {
	//fmt.Println("in app-router-condition.go func NewMphMatcherGroup")
	g := strmatcher.NewMphMatcherGroup()
	for _, d := range domains {
		matcherType, f := matcherTypeMap[d.Type]
		if !f {
			return nil, newError("unsupported domain type", d.Type)
		}
		_, err := g.AddPattern(d.Value, matcherType)
		if err != nil {
			return nil, err
		}
	}
	g.Build()
	return &DomainMatcher{
		matchers: g,
	}, nil
}

func NewDomainMatcher(domains []*Domain) (*DomainMatcher, error) {
	//fmt.Println("in app-router-condition.go func NewDomainMatcher")
	g := new(strmatcher.MatcherGroup)
	for _, d := range domains {
		m, err := domainToMatcher(d)
		if err != nil {
			return nil, err
		}
		g.Add(m)
	}

	return &DomainMatcher{
		matchers: g,
	}, nil
}

func (m *DomainMatcher) ApplyDomain(domain string) bool {
	return len(m.matchers.Match(strings.ToLower(domain))) > 0
}

// Apply implements Condition.
func (m *DomainMatcher) Apply(ctx routing.Context) bool {
	
	domain := ctx.GetTargetDomain()
	fmt.Println("in app-router-condition.go func (m *DomainMatcher) Apply domain is: ", domain)
	if len(domain) == 0 {
		return false
	}
	gb := m.ApplyDomain(domain)
	fmt.Println("in app-router-condition.go func (m *DomainMatcher) Apply gb is: ", gb)
	return gb
}

type MultiGeoIPMatcher struct {
	matchers []*GeoIPMatcher
	onSource bool
}

func NewMultiGeoIPMatcher(geoips []*GeoIP, onSource bool) (*MultiGeoIPMatcher, error) {
	//fmt.Println("in app-router-condition.go func NewMultiGeoIPMatcher")
	var matchers []*GeoIPMatcher
	for _, geoip := range geoips {
		matcher, err := globalGeoIPContainer.Add(geoip)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, matcher)
	}

	matcher := &MultiGeoIPMatcher{
		matchers: matchers,
		onSource: onSource,
	}

	return matcher, nil
}

// Apply implements Condition.
func (m *MultiGeoIPMatcher) Apply(ctx routing.Context) bool {
	fmt.Println("in app-router-condition.go func (m *MultiGeoIPMatcher) Apply")
	var ips []net.IP
	if m.onSource {
		ips = ctx.GetSourceIPs()
	} else {
		ips = ctx.GetTargetIPs()
	}
	for _, ip := range ips {
		for _, matcher := range m.matchers {
			if matcher.Match(ip) {
				return true
			}
		}
	}
	return false
}

type PortMatcher struct {
	port     net.MemoryPortList
	onSource bool
}

// NewPortMatcher create a new port matcher that can match source or destination port
func NewPortMatcher(list *net.PortList, onSource bool) *PortMatcher {
	//fmt.Println("in app-router-condition.go func NewPortMatcher")
	return &PortMatcher{
		port:     net.PortListFromProto(list),
		onSource: onSource,
	}
}

// Apply implements Condition.
func (v *PortMatcher) Apply(ctx routing.Context) bool {
	if v.onSource {
		return v.port.Contains(ctx.GetSourcePort())
	}
	return v.port.Contains(ctx.GetTargetPort())
}

type NetworkMatcher struct {
	list [8]bool
}

func NewNetworkMatcher(network []net.Network) NetworkMatcher {
	//fmt.Println("in app-router-condition.go func NewNetworkMatcher")
	var matcher NetworkMatcher
	for _, n := range network {
		matcher.list[int(n)] = true
	}
	return matcher
}

// Apply implements Condition.
func (v NetworkMatcher) Apply(ctx routing.Context) bool {
	return v.list[int(ctx.GetNetwork())]
}

type UserMatcher struct {
	user []string
}

func NewUserMatcher(users []string) *UserMatcher {
	//fmt.Println("in app-router-condition.go func NewUserMatcher")
	usersCopy := make([]string, 0, len(users))
	for _, user := range users {
		if len(user) > 0 {
			usersCopy = append(usersCopy, user)
		}
	}
	return &UserMatcher{
		user: usersCopy,
	}
}

// Apply implements Condition.
func (v *UserMatcher) Apply(ctx routing.Context) bool {
	fmt.Println("in app-router-condition.go func (v *UserMatcher) Apply")
	user := ctx.GetUser()
	if len(user) == 0 {
		return false
	}
	for _, u := range v.user {
		if u == user {
			return true
		}
	}
	return false
}

type InboundTagMatcher struct {
	tags []string
}

func NewInboundTagMatcher(tags []string) *InboundTagMatcher {
	//fmt.Println("in app-router-condition.go func NewInboundTagMatcher ")
	tagsCopy := make([]string, 0, len(tags))
	for _, tag := range tags {
		if len(tag) > 0 {
			tagsCopy = append(tagsCopy, tag)
		}
	}
	return &InboundTagMatcher{
		tags: tagsCopy,
	}
}

// Apply implements Condition.
func (v *InboundTagMatcher) Apply(ctx routing.Context) bool {
	fmt.Println("in app-router-condition.go func  (v *InboundTagMatcher) Apply ")
	tag := ctx.GetInboundTag()
	if len(tag) == 0 {
		return false
	}
	for _, t := range v.tags {
		if t == tag {
			return true
		}
	}
	return false
}

type ProtocolMatcher struct {
	protocols []string
}

func NewProtocolMatcher(protocols []string) *ProtocolMatcher {
	fmt.Println("in app-router-condition.go func NewProtocolMatcher")
	pCopy := make([]string, 0, len(protocols))

	for _, p := range protocols {
		if len(p) > 0 {
			pCopy = append(pCopy, p)
		}
	}

	return &ProtocolMatcher{
		protocols: pCopy,
	}
}

// Apply implements Condition.
func (m *ProtocolMatcher) Apply(ctx routing.Context) bool {
	fmt.Println("in app-router-condition.go func (m *ProtocolMatcher) Apply")
	protocol := ctx.GetProtocol()
	if len(protocol) == 0 {
		return false
	}
	for _, p := range m.protocols {
		if strings.HasPrefix(protocol, p) {
			return true
		}
	}
	return false
}

type AttributeMatcher struct {
	program *starlark.Program
}

func NewAttributeMatcher(code string) (*AttributeMatcher, error) {
	fmt.Println("in app-router-condition.go func NewAttributeMatcher")
	starFile, err := syntax.Parse("attr.star", "satisfied=("+code+")", 0)
	if err != nil {
		return nil, newError("attr rule").Base(err)
	}
	p, err := starlark.FileProgram(starFile, func(name string) bool {
		return name == "attrs"
	})
	if err != nil {
		return nil, err
	}
	return &AttributeMatcher{
		program: p,
	}, nil
}

// Match implements attributes matching.
func (m *AttributeMatcher) Match(attrs map[string]string) bool {
	fmt.Println("in app-router-condition.go func (m *AttributeMatcher) Match")
	attrsDict := new(starlark.Dict)
	for key, value := range attrs {
		attrsDict.SetKey(starlark.String(key), starlark.String(value))
	}

	predefined := make(starlark.StringDict)
	predefined["attrs"] = attrsDict

	thread := &starlark.Thread{
		Name: "matcher",
	}
	results, err := m.program.Init(thread, predefined)
	if err != nil {
		newError("attr matcher").Base(err).WriteToLog()
	}
	satisfied := results["satisfied"]
	return satisfied != nil && bool(satisfied.Truth())
}

// Apply implements Condition.
func (m *AttributeMatcher) Apply(ctx routing.Context) bool {
	fmt.Println("in app-router-condition.go func (m *AttributeMatcher) Apply")
	attributes := ctx.GetAttributes()
	if attributes == nil {
		return false
	}
	return m.Match(attributes)
}
