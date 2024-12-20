//go:build !confonly
// +build !confonly

package grpc

import (
	"context"
	gonet "net"
	"sync"
	"time"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/session"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/grpc/encoding"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/tls"
)

func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (internet.Connection, error) {
	fmt.Println("in transport-internet-grpc-dial.go func Dial")
	newError("creating connection to ", dest).WriteToLog(session.ExportIDToError(ctx))

	conn, err := dialgRPC(ctx, dest, streamSettings)
	if err != nil {
		return nil, newError("failed to dial Grpc").Base(err)
	}
	return internet.Connection(conn), nil
}

func init() {
	fmt.Println("in transport-internet-grpc-dial.go func init()")
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}

var (
	globalDialerMap    map[net.Destination]*grpc.ClientConn
	globalDialerAccess sync.Mutex
)

func dialgRPC(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (net.Conn, error) {
	fmt.Println("in transport-internet-grpc-dial.go func dialgRPC")
	grpcSettings := streamSettings.ProtocolSettings.(*Config)

	config := tls.ConfigFromStreamSettings(streamSettings)
	var dialOption = grpc.WithInsecure()

	if config != nil {
		dialOption = grpc.WithTransportCredentials(credentials.NewTLS(config.GetTLSConfig()))
	}

	conn, err := getGrpcClient(dest, dialOption)

	if err != nil {
		return nil, newError("Cannot dial grpc").Base(err)
	}
	client := encoding.NewGunServiceClient(conn)
	gunService, err := client.(encoding.GunServiceClientX).TunCustomName(ctx, grpcSettings.ServiceName)
	if err != nil {
		return nil, newError("Cannot dial grpc").Base(err)
	}
	return encoding.NewGunConn(gunService, nil), nil
}

func getGrpcClient(dest net.Destination, dialOption grpc.DialOption) (*grpc.ClientConn, error) {
	fmt.Println("in transport-internet-grpc-dial.go func getGrpcClient")
	globalDialerAccess.Lock()
	defer globalDialerAccess.Unlock()

	if globalDialerMap == nil {
		globalDialerMap = make(map[net.Destination]*grpc.ClientConn)
	}

	if client, found := globalDialerMap[dest]; found && client.GetState() != connectivity.Shutdown {
		return client, nil
	}

	conn, err := grpc.Dial(
		dest.Address.String()+":"+dest.Port.String(),
		dialOption,
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  500 * time.Millisecond,
				Multiplier: 1.5,
				Jitter:     0.2,
				MaxDelay:   19 * time.Second,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
		grpc.WithContextDialer(func(ctx context.Context, s string) (gonet.Conn, error) {
			rawHost, rawPort, err := net.SplitHostPort(s)
			if err != nil {
				return nil, err
			}
			if len(rawPort) == 0 {
				rawPort = "443"
			}
			port, err := net.PortFromString(rawPort)
			if err != nil {
				return nil, err
			}
			address := net.ParseAddress(rawHost)
			return internet.DialSystem(ctx, net.TCPDestination(address, port), nil)
		}),
	)
	globalDialerMap[dest] = conn
	return conn, err
}
