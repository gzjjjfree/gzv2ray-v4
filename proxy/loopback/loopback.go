//go:build !confonly
// +build !confonly

package loopback

import (
	"context"
	"fmt"

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/buf"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/retry"
	"github.com/gzjjjfree/gzv2ray-v4/common/session"
	"github.com/gzjjjfree/gzv2ray-v4/common/task"
	"github.com/gzjjjfree/gzv2ray-v4/features/routing"
	"github.com/gzjjjfree/gzv2ray-v4/transport"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
)

type Loopback struct {
	config             *Config
	dispatcherInstance routing.Dispatcher
}

func (l *Loopback) Process(ctx context.Context, link *transport.Link, _ internet.Dialer) error {
	fmt.Println("in proxy-loopback-looopback.go func (l *Loopback) Process")
	outbound := session.OutboundFromContext(ctx)
	if outbound == nil || !outbound.Target.IsValid() {
		return newError("target not specified.")
	}
	destination := outbound.Target

	newError("opening connection to ", destination).WriteToLog(session.ExportIDToError(ctx))

	input := link.Reader
	output := link.Writer

	var conn internet.Connection
	err := retry.ExponentialBackoff(5, 100).On(func() error {
		dialDest := destination

		content := new(session.Content)
		content.SkipDNSResolve = true

		ctx = session.ContextWithContent(ctx, content)

		inbound := session.InboundFromContext(ctx)

		inbound.Tag = l.config.InboundTag

		ctx = session.ContextWithInbound(ctx, inbound)

		rawConn, err := l.dispatcherInstance.Dispatch(ctx, dialDest)
		if err != nil {
			return err
		}

		var readerOpt net.ConnectionOption
		if dialDest.Network == net.Network_TCP {
			readerOpt = net.ConnectionOutputMulti(rawConn.Reader)
		} else {
			readerOpt = net.ConnectionOutputMultiUDP(rawConn.Reader)
		}

		conn = net.NewConnection(net.ConnectionInputMulti(rawConn.Writer), readerOpt)
		return nil
	})
	if err != nil {
		return newError("failed to open connection to ", destination).Base(err)
	}
	defer conn.Close()

	requestDone := func() error {
		var writer buf.Writer
		if destination.Network == net.Network_TCP {
			writer = buf.NewWriter(conn)
		} else {
			writer = &buf.SequentialWriter{Writer: conn}
		}

		if err := buf.Copy(input, writer); err != nil {
			return newError("failed to process request").Base(err)
		}

		return nil
	}

	responseDone := func() error {
		var reader buf.Reader
		if destination.Network == net.Network_TCP {
			reader = buf.NewReader(conn)
		} else {
			reader = buf.NewPacketReader(conn)
		}
		if err := buf.Copy(reader, output); err != nil {
			return newError("failed to process response").Base(err)
		}

		return nil
	}

	if err := task.Run(ctx, requestDone, task.OnSuccess(responseDone, task.Close(output))); err != nil {
		return newError("connection ends").Base(err)
	}

	return nil
}

func (l *Loopback) init(config *Config, dispatcherInstance routing.Dispatcher) error {
	fmt.Println("in proxy-loopback-looopback.go func (l *Loopback) init")
	l.dispatcherInstance = dispatcherInstance
	l.config = config
	return nil
}

func init() {
	fmt.Println("in proxy-loopback-looopback.go func  init()")
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		l := new(Loopback)
		err := core.RequireFeatures(ctx, func(dispatcherInstance routing.Dispatcher) error {
			return l.init(config.(*Config), dispatcherInstance)
		})
		return l, err
	}))
}
