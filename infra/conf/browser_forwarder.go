package conf

import (
	"google.golang.org/protobuf/proto"

	"github.com/gzjjjfree/gzv2ray-v4/app/browserforwarder"
)

type BrowserForwarderConfig struct {
	ListenAddr string `json:"listenAddr"`
	ListenPort int32  `json:"listenPort"`
}

func (b BrowserForwarderConfig) Build() (proto.Message, error) {
	return &browserforwarder.Config{
		ListenAddr: b.ListenAddr,
		ListenPort: b.ListenPort,
	}, nil
}
