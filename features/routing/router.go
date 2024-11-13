package routing

import (
	"fmt"
	"github.com/gzjjjfree/gzv2ray-v4/features"
	"github.com/gzjjjfree/gzv2ray-v4/common"
)


// RouterType return the type of Router interface. Can be used to implement common.HasType.
// RouterType 返回 Router 接口的类型。可用于实现 common.HasType。
// v2ray:api:stable
func RouterType() interface{} {
	return (*Router)(nil)
}

// Router is a feature to choose an outbound tag for the given request.
// 路由器是一种为给定请求选择出站标签的功能。
// v2ray:api:stable
type Router interface {
	features.Feature

	// PickRoute returns a route decision based on the given routing context.
	// PickRoute 根据给定的路由上下文返回路线决策。
	PickRoute(ctx Context) (Route, error)
}

// Route is the routing result of Router feature.
// Route 是 Router 功能的路由结果。
// v2ray:api:stable
type Route interface {
	// A Route is also a routing context.
	// Route 也是一个路由上下文。
	Context

	// GetOutboundGroupTags returns the detoured outbound group tags in sequence before a final outbound is chosen.
	// GetOutboundGroupTags 在选择最终出站之前按顺序返回绕行的出站组标签。
	GetOutboundGroupTags() []string

	// GetOutboundTag returns the tag of the outbound the connection was dispatched to.
	// GetOutboundTag 返回连接被调度到的出站标签。
	GetOutboundTag() string
}

// DefaultRouter is an implementation of Router, which always returns ErrNoClue for routing decisions.
// DefaultRouter 是 Router 的一个实现，它总是返回 ErrNoClue 用于路由决策。
type DefaultRouter struct{}

// Type implements common.HasType.
func (DefaultRouter) Type() interface{} {
	return RouterType()
}

// PickRoute implements Router.
func (DefaultRouter) PickRoute(ctx Context) (Route, error) {
	return nil, common.ErrNoClue
}

// Start implements common.Runnable.
func (DefaultRouter) Start() error {
	return nil
}

// Close implements common.Closable.
func (DefaultRouter) Close() error {
	fmt.Println("in features-routing-router.go func  (DefaultRouter) Close()")
	return nil
}
