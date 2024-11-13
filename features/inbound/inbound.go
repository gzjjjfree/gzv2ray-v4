package inbound

import (
	"context"
	"fmt"

	
	"github.com/gzjjjfree/gzv2ray-v4/features"
	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
)

// ManagerType returns the type of Manager interface. Can be used for implementing common.HasType.
// ManagerType 返回 Manager 接口的类型。可用于实现 common.HasType。
// v2ray:api:stable
func ManagerType() interface{} {
	fmt.Println("in features-inbound-inbound.go func ManagerType()")
	return (*Manager)(nil)
}

// Manager is a feature that manages InboundHandlers.
// Manager 是管理 InboundHandler 的功能。
// v2ray:api:stable
type Manager interface {
	features.Feature
	// GetHandlers returns an InboundHandler for the given tag.
	// GetHandlers 返回给定标签的 InboundHandler。
	GetHandler(ctx context.Context, tag string) (Handler, error)
	// AddHandler adds the given handler into this Manager.
	// AddHandler 将给定的处理程序添加到此管理器中。
	AddHandler(ctx context.Context, handler Handler) error

	// RemoveHandler removes a handler from Manager.
	// RemoveHandler 从 Manager 中删除一个处理程序。
	RemoveHandler(ctx context.Context, tag string) error
}

// Handler is the interface for handlers that process inbound connections.
// Handler 是处理入站连接的处理程序的接口。
// v2ray:api:stable
type Handler interface {
	common.Runnable
	// The tag of this handler.
	// 此处理程序的标签。
	Tag() string

	// Deprecated: Do not use in new code.
	// 已弃用：请勿在新代码中使用。
	GetRandomInboundProxy() (interface{}, net.Port, int)
}