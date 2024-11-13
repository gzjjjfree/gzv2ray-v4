package routing

import (
	"context"

	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/features"
	"github.com/gzjjjfree/gzv2ray-v4/transport"
)

// Dispatcher is a feature that dispatches inbound requests to outbound handlers based on rules.
// 调度程序是一种根据规则将入站请求调度到出站处理程序的功能。
// Dispatcher is required to be registered in a V2Ray instance to make V2Ray function properly.
// 需要在 V2Ray 实例中注册 Dispatcher 才能使 V2Ray 正常运行。
//
// v2ray:api:stable
type Dispatcher interface {
	features.Feature

	// Dispatch returns a Ray for transporting data for the given request.
	// Dispatch 返回一个用于传输给定请求的数据的 Ray
	Dispatch(ctx context.Context, dest net.Destination) (*transport.Link, error)
}

// DispatcherType returns the type of Dispatcher interface. Can be used to implement common.HasType.
// DispatcherType 返回 Dispatcher 接口的类型。可用于实现 common.HasType。
// v2ray:api:stable
func DispatcherType() interface{} {
	return (*Dispatcher)(nil)
}
