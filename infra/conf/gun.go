package conf

import (
	"google.golang.org/protobuf/proto"

	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/grpc"
)

type GunConfig struct {
	ServiceName string `json:"serviceName"`
}

func (g GunConfig) Build() (proto.Message, error) {
	return &grpc.Config{ServiceName: g.ServiceName}, nil
}
