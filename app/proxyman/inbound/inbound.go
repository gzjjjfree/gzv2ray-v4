package inbound

import (
	"context"
	"errors"
	"fmt"
	"sync"

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/app/proxyman"
	"github.com/gzjjjfree/gzv2ray-v4/common"

	//"github.com/gzjjjfree/gzv2ray-v4/common/serial"
	"github.com/gzjjjfree/gzv2ray-v4/common/session"
	"github.com/gzjjjfree/gzv2ray-v4/features/inbound"
)

// GetHandler implements inbound.Manager.
func (m *Manager) GetHandler(ctx context.Context, tag string) (inbound.Handler, error) {
	m.access.RLock()         // RLock 锁定 rw 以进行读取。
	defer m.access.RUnlock() // RUnlock 撤消单个 [RWMutex.RLock] 调用；它不会影响其他同时读取者。如果在进入 RUnlock 时 rw 未锁定以进行读取，则会出现运行时错误

	handler, found := m.taggedHandlers[tag]
	if !found {
		return nil, errors.New("handler not found")
	}
	return handler, nil
}

// Manager is to manage all inbound handlers.
// Manager 负责管理所有入站处理程序。
type Manager struct {
	access          sync.RWMutex
	untaggedHandler []inbound.Handler
	taggedHandlers  map[string]inbound.Handler
	running         bool
}

// AddHandler implements inbound.Manager.
func (m *Manager) AddHandler(ctx context.Context, handler inbound.Handler) error {
	m.access.Lock()
	defer m.access.Unlock()

	tag := handler.Tag()
	if len(tag) > 0 {
		m.taggedHandlers[tag] = handler // 设置 taggedHandlers 中字符键 tag 值为 handler
	} else {
		m.untaggedHandler = append(m.untaggedHandler, handler)
	}

	if m.running {
		return handler.Start()
	}

	return nil
}

// RemoveHandler implements inbound.Manager.
func (m *Manager) RemoveHandler(ctx context.Context, tag string) error {
	fmt.Println("in app-proxyman-inbound-inbound.go func (m *Manager) RemoveHandler")
	if tag == "" {
		return common.ErrNoClue
	}

	m.access.Lock()
	defer m.access.Unlock()

	if handler, found := m.taggedHandlers[tag]; found {
		fmt.Println("in app-proxyman-inbound-inbound.go func (m *Manager) RemoveHandler if err := handler.Close()")
		if err := handler.Close(); err != nil {
			fmt.Println("failed to close handler")
		}
		delete(m.taggedHandlers, tag)
		return nil
	}

	return common.ErrNoClue
}

// Start implements common.Runnable.
func (m *Manager) Start() error {
	fmt.Println("in app-proxyman-inbound-inbound.go func Start()")
	m.access.Lock()
	defer m.access.Unlock()

	m.running = true

	for _, handler := range m.taggedHandlers {
		if err := handler.Start(); err != nil {
			return err
		}
	}

	for _, handler := range m.untaggedHandler {
		if err := handler.Start(); err != nil {
			return err
		}
	}
	return nil
}

// Close implements common.Closable.
func (m *Manager) Close() error {
	fmt.Println("in app-proxyman-inbound-inbound.go func (m *Manager) Close()")
	m.access.Lock()
	defer m.access.Unlock()

	m.running = false

	var errorsmsg []interface{}
	for _, handler := range m.taggedHandlers {
		if err := handler.Close(); err != nil {
			errorsmsg = append(errorsmsg, err)
		}
	}
	for _, handler := range m.untaggedHandler {
		if err := handler.Close(); err != nil {
			errorsmsg = append(errorsmsg, err)
		}
	}

	if len(errorsmsg) > 0 {
		return errors.New("failed to close all handlers")
	}

	return nil
}

// New returns a new Manager for inbound handlers.
// New 返回一个新的入站处理程序的管理器。
func New(ctx context.Context, config *proxyman.InboundConfig) (*Manager, error) {
	m := &Manager{
		taggedHandlers: make(map[string]inbound.Handler),
	}
	return m, nil
}

// NewHandler creates a new inbound.Handler based on the given config.
// NewHandler 根据给定的配置创建一个新的 inbound.Handler。
func NewHandler(ctx context.Context, config *core.InboundHandlerConfig) (inbound.Handler, error) {
	rawReceiverSettings, err := config.ReceiverSettings.GetInstance()
	if err != nil {
		return nil, err
	}
	proxySettings, err := config.ProxySettings.GetInstance()
	if err != nil {
		return nil, err
	}
	tag := config.Tag

	receiverSettings, ok := rawReceiverSettings.(*proxyman.ReceiverConfig)
	if !ok {
		return nil, errors.New("not a ReceiverConfig")
	}

	streamSettings := receiverSettings.StreamSettings
	if streamSettings != nil && streamSettings.SocketSettings != nil {
		ctx = session.ContextWithSockopt(ctx, &session.Sockopt{
			Mark: streamSettings.SocketSettings.Mark,
		})
	}

	allocStrategy := receiverSettings.AllocationStrategy
	if allocStrategy == nil || allocStrategy.Type == proxyman.AllocationStrategy_Always {
		return NewAlwaysOnInboundHandler(ctx, tag, receiverSettings, proxySettings)
	}

	if allocStrategy.Type == proxyman.AllocationStrategy_Random {
		return NewDynamicInboundHandler(ctx, tag, receiverSettings, proxySettings)
	}
	return nil, errors.New("unknown allocation strategy: ")
}


// Type implements common.HasType.
func (*Manager) Type() interface{} {
	return inbound.ManagerType()
}


func init() {
	fmt.Println("is run ./app/proxyman/inbound/inbound.go func init ")
	common.Must(common.RegisterConfig((*proxyman.InboundConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return New(ctx, config.(*proxyman.InboundConfig))
	}))
	common.Must(common.RegisterConfig((*core.InboundHandlerConfig)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		return NewHandler(ctx, config.(*core.InboundHandlerConfig))
	}))
}
