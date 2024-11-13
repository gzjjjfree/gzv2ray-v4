package tagged

import (
	"context"

	"github.com/gzjjjfree/gzv2ray-v4/common/net"
)

type DialFunc func(ctx context.Context, dest net.Destination, tag string) (net.Conn, error)

var Dialer DialFunc
