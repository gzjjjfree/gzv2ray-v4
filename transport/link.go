package transport

import "github.com/gzjjjfree/gzv2ray-v4/common/buf"

// Link is a utility for connecting between an inbound and an outbound proxy handler.
// Link 是一个用于连接入站和出站代理处理程序的实用程序
type Link struct {
	Reader buf.Reader
	Writer buf.Writer
}
