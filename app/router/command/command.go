// +build !confonly

package command



import (
	"context"
	"time"
	"fmt"

	"google.golang.org/grpc"

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/features/routing"
	"github.com/gzjjjfree/gzv2ray-v4/features/stats"
)

// routingServer is an implementation of RoutingService.
type routingServer struct {
	router       routing.Router
	routingStats stats.Channel
}

// NewRoutingServer creates a statistics service with statistics manager.
func NewRoutingServer(router routing.Router, routingStats stats.Channel) RoutingServiceServer {
	fmt.Println("in app-router-command-command.go func NewRoutingServer")
	return &routingServer{
		router:       router,
		routingStats: routingStats,
	}
}

func (s *routingServer) TestRoute(ctx context.Context, request *TestRouteRequest) (*RoutingContext, error) {
	fmt.Println("in app-router-command-command.go func (s *routingServer) TestRoute")
	if request.RoutingContext == nil {
		return nil, newError("Invalid routing request.")
	}
	route, err := s.router.PickRoute(AsRoutingContext(request.RoutingContext))
	if err != nil {
		return nil, err
	}
	if request.PublishResult && s.routingStats != nil {
		ctx, _ := context.WithTimeout(context.Background(), 4*time.Second) // nolint: govet
		s.routingStats.Publish(ctx, route)
	}
	return AsProtobufMessage(request.FieldSelectors)(route), nil
}

func (s *routingServer) SubscribeRoutingStats(request *SubscribeRoutingStatsRequest, stream RoutingService_SubscribeRoutingStatsServer) error {
	fmt.Println("in app-router-command-command.go func (s *routingServer) SubscribeRoutingStats")
	if s.routingStats == nil {
		return newError("Routing statistics not enabled.")
	}
	genMessage := AsProtobufMessage(request.FieldSelectors)
	subscriber, err := stats.SubscribeRunnableChannel(s.routingStats)
	if err != nil {
		return err
	}
	defer stats.UnsubscribeClosableChannel(s.routingStats, subscriber)
	for {
		select {
		case value, ok := <-subscriber:
			if !ok {
				return newError("Upstream closed the subscriber channel.")
			}
			route, ok := value.(routing.Route)
			if !ok {
				return newError("Upstream sent malformed statistics.")
			}
			err := stream.Send(genMessage(route))
			if err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *routingServer) mustEmbedUnimplementedRoutingServiceServer() {}

type service struct {
	v *core.Instance
}

func (s *service) Register(server *grpc.Server) {
	common.Must(s.v.RequireFeatures(func(router routing.Router, stats stats.Manager) {
		RegisterRoutingServiceServer(server, NewRoutingServer(router, nil))
	}))
}

func init() {
	fmt.Println("in app-router-command-command.go func init()")
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, cfg interface{}) (interface{}, error) {
		s := core.MustFromContext(ctx)
		return &service{v: s}, nil
	}))
}
