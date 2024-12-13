package policy

import (
	"time"
	"runtime"
	"context"
	//"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/features"
	"github.com/gzjjjfree/gzv2ray-v4/common/platform"
)

// ManagerType returns the type of Manager interface. Can be used to implement common.HasType.
// ManagerType 返回 Manager 接口的类型。可用于实现 common.HasType。
// v2ray:api:stable
func ManagerType() interface{} {
	return (*Manager)(nil)
}

// Manager is a feature that provides Policy for the given user by its id or level.
// 管理器是一项根据给定用户的 ID 或级别提供策略的功能。
// v2ray:api:stable
type Manager interface {
	features.Feature

	// ForLevel returns the Session policy for the given user level.
	// ForLevel 返回给定用户级别的会话策略。
	ForLevel(level uint32) Session

	// ForSystem returns the System policy for V2Ray system.
	// ForSystem 返回 V2Ray 系统的系统策略。
	ForSystem() System
}

// System contains policy settings at system level.
// 系统包含系统级别的策略设置。
type System struct {
	Stats  SystemStats
	Buffer Buffer
}

// Session is session based settings for controlling V2Ray requests. It contains various settings (or limits) that may differ for different users in the context.
// Session 是用于控制 V2Ray 请求的基于会话的设置。它包含各种设置（或限制），这些设置（或限制）可能因上下文中的不同用户而异。
type Session struct {
	Timeouts Timeout // Timeout settings
	Stats    Stats
	Buffer   Buffer
}

// Timeout contains limits for connection timeout.
// 超时包含连接超时的限制。
type Timeout struct {
	// Timeout for handshake phase in a connection.
	// 连接中的握手阶段超时。
	Handshake time.Duration
	// Timeout for connection being idle, i.e., there is no egress or ingress traffic in this connection.
	// 连接空闲超时，即此连接中没有出站或入站流量。
	ConnectionIdle time.Duration
	// Timeout for an uplink only connection, i.e., the downlink of the connection has been closed.
	// 仅上行链路连接超时，即连接的下行链路已关闭。
	UplinkOnly time.Duration
	// Timeout for an downlink only connection, i.e., the uplink of the connection has been closed.
	// 仅下行链路连接超时，即连接的上行链路已关闭。
	DownlinkOnly time.Duration
}

// Stats contains settings for stats counters.
// 统计数据包含统计计数器的设置。
type Stats struct {
	// Whether or not to enable stat counter for user uplink traffic.
	// 是否启用用户上行流量统计计数器。
	UserUplink bool
	// Whether or not to enable stat counter for user downlink traffic.
	// 是否启用用户下行流量统计计数器。
	UserDownlink bool
}

// Buffer contains settings for internal buffer.
// 缓冲区包含内部缓冲区的设置。
type Buffer struct {
	// Size of buffer per connection, in bytes. -1 for unlimited buffer.
	// 每个连接的缓冲区大小（以字节为单位）。-1 表示无限制缓冲区。
	PerConnection int32
}

// SystemStats contains stat policy settings on system level.
// SystemStats 包含系统级别的统计策略设置。
type SystemStats struct {
	// Whether or not to enable stat counter for uplink traffic in inbound handlers.
	// 是否在入站处理程序中启用上行链路流量的统计计数器。	
	InboundUplink bool
	// Whether or not to enable stat counter for downlink traffic in inbound handlers.
	// 是否在入站处理程序中启用下行流量统计计数器。
	InboundDownlink bool
	// Whether or not to enable stat counter for uplink traffic in outbound handlers.
	// 是否在出站处理程序中启用上行链路流量的统计计数器。
	OutboundUplink bool
	// Whether or not to enable stat counter for downlink traffic in outbound handlers.
	// 是否在出站处理程序中启用下行流量统计计数器。
	OutboundDownlink bool
}

func defaultBufferPolicy() Buffer {
	return Buffer{
		PerConnection: defaultBufferSize,
	}
}

var defaultBufferSize int32


func init() {
	const key = "gzv2ray.ray.buffer.size"
	const defaultValue = -17
	size := platform.EnvFlag{
		Name:    key,
		AltName: platform.NormalizeEnvName(key),
	}.GetValueAsInt(defaultValue)

	switch size {
	case 0:
		defaultBufferSize = -1 // For pipe to use unlimited size
	case defaultValue: // Env flag not defined. Use default values per CPU-arch.
		switch runtime.GOARCH {
		case "arm", "mips", "mipsle":
			defaultBufferSize = 0
		case "arm64", "mips64", "mips64le":
			defaultBufferSize = 4 * 1024 // 4k cache for low-end devices
		default:
			defaultBufferSize = 512 * 1024
		}
	default:
		defaultBufferSize = int32(size) * 1024 * 1024
	}
}

// SessionDefault returns the Policy when user is not specified.
// 当未指定用户时，SessionDefault 返回 Policy
func SessionDefault() Session {
	//fmt.Println("in features-policy-policy.go func SessionDefault()")
	return Session{
		Timeouts: Timeout{
			// Align Handshake timeout with nginx client_header_timeout
			// So that this value will not indicate server identity
			// 将握手超时与 nginx client_header_timeout 对齐, 这样该值就不会表明服务器身份
			Handshake:      time.Second * 60,
			ConnectionIdle: time.Second * 300,
			UplinkOnly:     time.Second * 1,
			DownlinkOnly:   time.Second * 1,
		},
		Stats: Stats{
			UserUplink:   false,
			UserDownlink: false,
		},
		Buffer: defaultBufferPolicy(),
	}
}

type policyKey int32

const (
	bufferPolicyKey policyKey = 0
)

func ContextWithBufferPolicy(ctx context.Context, p Buffer) context.Context {
	return context.WithValue(ctx, bufferPolicyKey, p)
}

func BufferPolicyFromContext(ctx context.Context) Buffer {
	//fmt.Println("in features-policy-policy.go func BufferPolicyFromContext ctx: ", ctx)
	pPolicy := ctx.Value(bufferPolicyKey)
	//fmt.Println("in features-policy-policy.go func BufferPolicyFromContext pPolicy: ", pPolicy)
	if pPolicy == nil {
		//fmt.Println("in features-policy-policy.go func BufferPolicyFromContext pPolicy == nil")
		return defaultBufferPolicy()
	}
	return pPolicy.(Buffer)
}
