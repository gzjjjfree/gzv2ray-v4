package routing

import (
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
)

// Context is a feature to store connection information for routing.
// 上下文是用于存储路由的连接信息的功能。
// v2ray:api:stable
type Context interface {
	// GetInboundTag returns the tag of the inbound the connection was from.
	// GetInboundTag 返回连接来自的入站标签。
	GetInboundTag() string

	// GetSourcesIPs returns the source IPs bound to the connection.
	// GetSourcesIPs 返回与连接绑定的源 IP。
	GetSourceIPs() []net.IP

	// GetSourcePort returns the source port of the connection.
	// GetSourcePort 返回连接的源端口。
	GetSourcePort() net.Port

	// GetTargetIPs returns the target IP of the connection or resolved IPs of target domain.
	// GetTargetIPs 返回连接的目标 IP 或目标域的解析 IP。
	GetTargetIPs() []net.IP

	// GetTargetPort returns the target port of the connection.
	// GetTargetPort 返回连接的目标端口。
	GetTargetPort() net.Port

	// GetTargetDomain returns the target domain of the connection, if exists.
	// 如果存在，GetTargetDomain 将返回连接的目标域。
	GetTargetDomain() string

	// GetNetwork returns the network type of the connection.
	// GetNetwork 返回连接的网络类型。
	GetNetwork() net.Network

	// GetProtocol returns the protocol from the connection content, if sniffed out.
	// 如果嗅探到的话，GetProtocol 将返回连接内容中的协议。
	GetProtocol() string

	// GetUser returns the user email from the connection content, if exists.
	// 如果存在，GetUser 将返回连接内容中的用户电子邮件。
	GetUser() string

	// GetAttributes returns extra attributes from the conneciont content.
	// GetAttributes 从连接内容中返回额外的属性。
	GetAttributes() map[string]string

	// GetSkipDNSResolve returns a flag switch for weather skip dns resolve during route pick.
	// GetSkipDNSResolve 返回一个标志开关，用于在路线选择期间跳过 DNS 解析。
	GetSkipDNSResolve() bool
}