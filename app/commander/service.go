// +build !confonly

package commander

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/gzjjjfree/gzv2ray-v4/common"
)

// Service is a Commander service.
type Service interface {
	// Register registers the service itself to a gRPC server.
	Register(*grpc.Server)
}

type reflectionService struct{}

func (r reflectionService) Register(s *grpc.Server) {
	reflection.Register(s)
}

func init() {
	fmt.Println("is run ./app/commander/service.go func init ")
	common.Must(common.RegisterConfig((*ReflectionConfig)(nil), func(ctx context.Context, cfg interface{}) (interface{}, error) {
		return reflectionService{}, nil
	}))
}
