// Package proxy contains all proxies used by V2Ray.
// 代理包包含 V2Ray 使用的所有代理
// To implement an inbound or outbound proxy, one needs to do the following:
// 要实现入站或出站代理，需要执行以下操作
// 1. Implement the interface(s) below.
// 1. 实现下面的接口。
// 2. Register a config creator through common.RegisterConfig.
// 2.通过common.RegisterConfig注册一个配置创建器。
package proxy

import (
	"context"

	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/protocol"
	"github.com/gzjjjfree/gzv2ray-v4/features/routing"
	"github.com/gzjjjfree/gzv2ray-v4/transport"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
)

// An Inbound processes inbound connections.
// 入站处理入站连接。
type Inbound interface {
	// Network returns a list of networks that this inbound supports. Connections with not-supported networks will not be passed into Process().
	// Network 返回此入站支持的网络列表。不支持的网络的连接将不会传递到 Process() 中。
	Network() []net.Network

	// Process processes a connection of given network. If necessary, the Inbound can dispatch the connection to an Outbound.
	// 进程处理给定网络的连接。如有必要，入站可以将连接调度到出站。
	Process(context.Context, net.Network, internet.Connection, routing.Dispatcher) error
}


// An Outbound process outbound connections.
type Outbound interface {
	// Process processes the given connection. The given dialer may be used to dial a system outbound connection.
	Process(context.Context, *transport.Link, internet.Dialer) error
}

// UserManager is the interface for Inbounds and Outbounds that can manage their users.
type UserManager interface {
	// AddUser adds a new user.
	AddUser(context.Context, *protocol.MemoryUser) error

	// RemoveUser removes a user by email.
	RemoveUser(context.Context, string) error
}

type GetInbound interface {
	GetInbound() Inbound
}

type GetOutbound interface {
	GetOutbound() Outbound
}
