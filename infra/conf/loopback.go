package conf

import (
	"google.golang.org/protobuf/proto"

	"github.com/gzjjjfree/gzv2ray-v4/proxy/loopback"
)

type LoopbackConfig struct {
	InboundTag string `json:"inboundTag"`
}

func (l LoopbackConfig) Build() (proto.Message, error) {
	return &loopback.Config{InboundTag: l.InboundTag}, nil
}
