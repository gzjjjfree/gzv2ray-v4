//go:build !confonly
// +build !confonly

package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/session"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/grpc/encoding"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/tls"
)

type Listener struct {
	encoding.UnimplementedGunServiceServer
	ctx     context.Context
	handler internet.ConnHandler
	local   net.Addr
	config  *Config
	locker  *internet.FileLocker // for unix domain socket

	s *grpc.Server
}

func (l Listener) Tun(server encoding.GunService_TunServer) error {
	fmt.Println("in transport-internet-grpc-hub.go func (l Listener) Tun")
	tunCtx, cancel := context.WithCancel(l.ctx)
	l.handler(encoding.NewGunConn(server, cancel))
	<-tunCtx.Done()
	return nil
}

func (l Listener) Close() error {
	fmt.Println("in transport-internet-grpc-hub.go func (l Listener) Close()")
	l.s.Stop()
	return nil
}

func (l Listener) Addr() net.Addr {
	return l.local
}

func Listen(ctx context.Context, address net.Address, port net.Port, settings *internet.MemoryStreamConfig, handler internet.ConnHandler) (internet.Listener, error) {
	fmt.Println("in transport-internet-grpc-hub.go func Listen")
	grpcSettings := settings.ProtocolSettings.(*Config)
	var listener *Listener
	if port == net.Port(0) { // unix
		listener = &Listener{
			handler: handler,
			local: &net.UnixAddr{
				Name: address.Domain(),
				Net:  "unix",
			},
			config: grpcSettings,
		}
	} else { // tcp
		listener = &Listener{
			handler: handler,
			local: &net.TCPAddr{
				IP:   address.IP(),
				Port: int(port),
			},
			config: grpcSettings,
		}
	}

	listener.ctx = ctx

	config := tls.ConfigFromStreamSettings(settings)

	var s *grpc.Server
	if config == nil {
		s = grpc.NewServer()
	} else {
		s = grpc.NewServer(grpc.Creds(credentials.NewTLS(config.GetTLSConfig(tls.WithNextProto("h2")))))
	}
	listener.s = s

	if settings.SocketSettings != nil && settings.SocketSettings.AcceptProxyProtocol {
		newError("accepting PROXY protocol").AtWarning().WriteToLog(session.ExportIDToError(ctx))
	}

	go func() {
		var streamListener net.Listener
		var err error
		if port == net.Port(0) { // unix
			streamListener, err = internet.ListenSystem(ctx, &net.UnixAddr{
				Name: address.Domain(),
				Net:  "unix",
			}, settings.SocketSettings)
			if err != nil {
				newError("failed to listen on ", address).Base(err).AtError().WriteToLog(session.ExportIDToError(ctx))
				return
			}
			locker := ctx.Value(address.Domain())
			if locker != nil {
				listener.locker = locker.(*internet.FileLocker)
			}
		} else { // tcp
			streamListener, err = internet.ListenSystem(ctx, &net.TCPAddr{
				IP:   address.IP(),
				Port: int(port),
			}, settings.SocketSettings)
			if err != nil {
				newError("failed to listen on ", address, ":", port).Base(err).AtError().WriteToLog(session.ExportIDToError(ctx))
				return
			}
		}

		encoding.RegisterGunServiceServerX(s, listener, grpcSettings.ServiceName)

		if err = s.Serve(streamListener); err != nil {
			newError("Listener for grpc ended").Base(err).WriteToLog()
		}
	}()

	return listener, nil
}

func init() {
	common.Must(internet.RegisterTransportListener(protocolName, Listen))
}