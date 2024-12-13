//go:build !confonly
// +build !confonly

package websocket

import (
	"net/http"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
)

const protocolName = "websocket"

func (c *Config) GetNormalizedPath() string {
	fmt.Println("in transport-internet-websocket config.go func (c *Config) GetNormalizedPath()")
	path := c.Path
	if path == "" {
		return "/"
	}
	if path[0] != '/' {
		return "/" + path
	}
	return path
}

func (c *Config) GetRequestHeader() http.Header {
	header := http.Header{}
	for _, h := range c.Header {
		header.Add(h.Key, h.Value)
	}
	return header
}

func init() {
	common.Must(internet.RegisterProtocolConfigCreator(protocolName, func() interface{} {
		return new(Config)
	}))
}
