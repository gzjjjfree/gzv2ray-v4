package stats

import (
	"context"
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/features"
)

// ManagerType returns the type of Manager interface. Can be used to implement common.HasType.
// ManagerType 返回 Manager 接口的类型。可用于实现 common.HasType。
// v2ray:api:stable
func ManagerType() interface{} {
	return (*Manager)(nil)
}

// NoopManager is an implementation of Manager, which doesn't has actual functionalities.
// NoopManager 是 Manager 的一个实现，它没有实际的功能。
type NoopManager struct{}

// Manager is the interface for stats manager.
// Manager 是统计管理器的界面。
// v2ray:api:stable
type Manager interface {
	features.Feature

	// RegisterCounter registers a new counter to the manager. The identifier string must not be empty, and unique among other counters.
	// RegisterCounter 向管理器注册一个新的计数器。标识符字符串不能为空，且在其他计数器中是唯一的。
	RegisterCounter(string) (Counter, error)
	// UnregisterCounter unregisters a counter from the manager by its identifier.
	// UnregisterCounter 通过其标识符从管理器中取消注册计数器。
	UnregisterCounter(string) error
	// GetCounter returns a counter by its identifier.
	// GetCounter 通过其标识符返回一个计数器。
	GetCounter(string) Counter

	// RegisterChannel registers a new channel to the manager. The identifier string must not be empty, and unique among other channels.
	// RegisterChannel 向管理器注册一个新通道。标识符字符串不能为空，且与其他通道不同。
	RegisterChannel(string) (Channel, error)
	// UnregisterCounter unregisters a channel from the manager by its identifier.
	// UnregisterCounter 根据通道标识符从管理器中取消注册该通道。
	UnregisterChannel(string) error
	// GetChannel returns a channel by its identifier.
	// GetChannel 通过其标识符返回一个通道。
	GetChannel(string) Channel
}

// Channel is the interface for stats channel.
// Channel 是统计通道的接口。
// v2ray:api:stable
type Channel interface {
	// Channel is a runnable unit.
	// 通道是一个可运行的单元。
	common.Runnable
	// Publish broadcasts a message through the channel with a controlling context.
	// 发布通过具有控制上下文的频道广播消息。
	Publish(context.Context, interface{})
	// SubscriberCount returns the number of the subscribers.
	// SubscriberCount 返回订阅者的数量。
	Subscribers() []chan interface{}
	// Subscribe registers for listening to channel stream and returns a new listener channel.
	// 订阅注册以监听频道流并返回一个新的监听频道。
	Subscribe() (chan interface{}, error)
	// Unsubscribe unregisters a listener channel from current Channel object.
	// 取消订阅将从当前 Channel 对象中取消注册侦听器通道。
	Unsubscribe(chan interface{}) error
}

// Counter is the interface for stats counters.
// Counter 是统计计数器的接口。
// v2ray:api:stable
type Counter interface {
	// Value is the current value of the counter.
	// 值是计数器的当前值。
	Value() int64
	// Set sets a new value to the counter, and returns the previous one.
	// Set 为计数器设置一个新值，并返回前一个值。
	Set(int64) int64
	// Add adds a value to the current counter value, and returns the previous value.
	// Add 将一个值添加到当前计数器值，并返回先前的值。
	Add(int64) int64
}

// SubscribeRunnableChannel subscribes the channel and starts it if there is first subscriber coming.
func SubscribeRunnableChannel(c Channel) (chan interface{}, error) {
	if len(c.Subscribers()) == 0 {
		if err := c.Start(); err != nil {
			return nil, err
		}
	}
	return c.Subscribe()
}

// UnsubscribeClosableChannel unsubcribes the channel and close it if there is no more subscriber.
// 如果没有更多订阅者，UnsubscribeClosableChannel 将取消订阅该频道并将其关闭。
func UnsubscribeClosableChannel(c Channel, sub chan interface{}) error {
	fmt.Println("in features-stats-stats.go func UnsubscribeClosableChannel")
	if err := c.Unsubscribe(sub); err != nil {
		return err
	}
	if len(c.Subscribers()) == 0 {
		fmt.Println("in features-stats-stats.go func UnsubscribeClosableChannel return c.Close()")
		return c.Close()
	}
	return nil
}

// GetOrRegisterCounter tries to get the StatCounter first. If not exist, it then tries to create a new counter.
func GetOrRegisterCounter(m Manager, name string) (Counter, error) {
	counter := m.GetCounter(name)
	if counter != nil {
		return counter, nil
	}

	return m.RegisterCounter(name)
}

// GetOrRegisterChannel tries to get the StatChannel first. If not exist, it then tries to create a new channel.
func GetOrRegisterChannel(m Manager, name string) (Channel, error) {
	channel := m.GetChannel(name)
	if channel != nil {
		return channel, nil
	}

	return m.RegisterChannel(name)
}

// Type implements common.HasType.
func (NoopManager) Type() interface{} {
	return ManagerType()
}

// RegisterCounter implements Manager.
func (NoopManager) RegisterCounter(string) (Counter, error) {
	return nil, errors.New("not implemented")
}

// UnregisterCounter implements Manager.
func (NoopManager) UnregisterCounter(string) error {
	return nil
}

// GetCounter implements Manager.
func (NoopManager) GetCounter(string) Counter {
	return nil
}

// RegisterChannel implements Manager.
func (NoopManager) RegisterChannel(string) (Channel, error) {
	return nil, errors.New("not implemented")
}

// UnregisterChannel implements Manager.
func (NoopManager) UnregisterChannel(string) error {
	return nil
}

// GetChannel implements Manager.
func (NoopManager) GetChannel(string) Channel {
	return nil
}

// Start implements common.Runnable.
func (NoopManager) Start() error { return nil }

// Close implements common.Closable.
func (NoopManager) Close() error { return nil }
