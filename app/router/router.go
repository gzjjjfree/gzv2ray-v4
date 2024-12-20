// +build !confonly

package router



import (
	"context"
	"fmt"

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/features/dns"
	"github.com/gzjjjfree/gzv2ray-v4/features/outbound"
	"github.com/gzjjjfree/gzv2ray-v4/features/routing"
	routing_dns "github.com/gzjjjfree/gzv2ray-v4/features/routing/dns"
)

// Router is an implementation of routing.Router.
type Router struct {
	domainStrategy Config_DomainStrategy
	rules          []*Rule
	balancers      map[string]*Balancer
	dns            dns.Client
}

// Route is an implementation of routing.Route.
type Route struct {
	routing.Context
	outboundGroupTags []string
	outboundTag       string
}

// Init initializes the Router.
func (r *Router) Init(config *Config, d dns.Client, ohm outbound.Manager) error {
	fmt.Println("in app-router-router.go func (r *Router) Init")
	r.domainStrategy = config.DomainStrategy
	r.dns = d

	r.balancers = make(map[string]*Balancer, len(config.BalancingRule))
	for _, rule := range config.BalancingRule {
		balancer, err := rule.Build(ohm)
		if err != nil {
			return err
		}
		r.balancers[rule.Tag] = balancer
	}

	r.rules = make([]*Rule, 0, len(config.Rule))
	for _, rule := range config.Rule {
		cond, err := rule.BuildCondition()
		if err != nil {
			return err
		}
		rr := &Rule{
			Condition: cond,
			Tag:       rule.GetTag(),
		}
		btag := rule.GetBalancingTag()
		if len(btag) > 0 {
			brule, found := r.balancers[btag]
			if !found {
				return newError("balancer ", btag, " not found")
			}
			rr.Balancer = brule
		}
		r.rules = append(r.rules, rr)
	}

	return nil
}

// PickRoute implements routing.Router.
func (r *Router) PickRoute(ctx routing.Context) (routing.Route, error) {
	fmt.Println("in app-router-router.go func (r *Router) PickRoute")
	rule, ctx, err := r.pickRouteInternal(ctx)
	if err != nil {
		return nil, err
	}
	tag, err := rule.GetTag()
	fmt.Println("in app-router-router.go func (r *Router) PickRoute tag is: ", tag)
	if err != nil {
		return nil, err
	}
	return &Route{Context: ctx, outboundTag: tag}, nil
}

func (r *Router) pickRouteInternal(ctx routing.Context) (*Rule, routing.Context, error) {
	fmt.Println("in app-router-router.go func (r *Router) pickRouteInternal GetTargetIPs(): ", ctx.GetTargetIPs(), " GetTargetDomain(): ", ctx.GetTargetDomain())
	// SkipDNSResolve is set from DNS module.
	// the DOH remote server maybe a domain name,
	// this prevents cycle resolving dead loop
	skipDNSResolve := ctx.GetSkipDNSResolve()
	fmt.Println("in app-router-router.go func (r *Router) pickRouteInternal skipDNSResolve is: ", skipDNSResolve)
	if r.domainStrategy == Config_IpOnDemand && !skipDNSResolve {
		fmt.Println("in app-router-router.go func (r *Router) pickRouteInternal !skipDNSResolve")
		ctx = routing_dns.ContextWithDNSClient(ctx, r.dns)
	}
	fmt.Println("in app-router-router.go func (r *Router) pickRouteInternal 匹配查找域名？")
	// 根据路由规则匹配域名确定出站 tag
	for _, rule := range r.rules {
		if rule.Apply(ctx) {
			return rule, ctx, nil
		}
	}

	// 当域名不匹配或config 设置不是遇到 IP 规则请求 DNS 或没有 DNS 设置时
	if r.domainStrategy != Config_IpIfNonMatch || len(ctx.GetTargetDomain()) == 0 || skipDNSResolve {
		fmt.Println("in app-router-router.go func (r *Router) pickRouteInternal len(ctx.GetTargetDomain()) == 0")
		return nil, ctx, common.ErrNoClue
	}

	ctx = routing_dns.ContextWithDNSClient(ctx, r.dns)
// 根据路由规则匹配 IP 确定出站 tag
	fmt.Println("in app-router-router.go func (r *Router) pickRouteInternal 匹配查找IP？")
	// Try applying rules again if we have IPs.
	for _, rule := range r.rules {
		if rule.Apply(ctx) {
			return rule, ctx, nil
		}
	}

	return nil, ctx, common.ErrNoClue
}

// Start implements common.Runnable.
func (*Router) Start() error {
	fmt.Println("in  app-router-router.go  func Start()")
	return nil
}

// Close implements common.Closable.
func (*Router) Close() error {
	fmt.Println("in app-router-router.go func (*Router) Close()")
	return nil
}

// Type implement common.HasType.
func (*Router) Type() interface{} {
	return routing.RouterType()
}

// GetOutboundGroupTags implements routing.Route.
func (r *Route) GetOutboundGroupTags() []string {
	return r.outboundGroupTags
}

// GetOutboundTag implements routing.Route.
func (r *Route) GetOutboundTag() string {
	return r.outboundTag
}

func init() {
	fmt.Println("in app-router-router.go func init()")
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		r := new(Router)
		if err := core.RequireFeatures(ctx, func(d dns.Client, ohm outbound.Manager) error {
			return r.Init(config.(*Config), d, ohm)
		}); err != nil {
			return nil, err
		}
		return r, nil
	}))
}
