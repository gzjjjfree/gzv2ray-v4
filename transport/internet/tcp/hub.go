// +build !confonly

package tcp

import (
	"context"
	gotls "crypto/tls"
	"strings"
	"time"
	"fmt"
	"errors"

	//"example.com/gztest"
	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	//"github.com/gzjjjfree/gzv2ray-v4/common/session"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/tls"
)

// Listener is an internet.Listener that listens for TCP connections.
// Listener 是一个用于监听 TCP 连接的 internet.Listener
type Listener struct {
	listener   net.Listener
	tlsConfig  *gotls.Config
	authConfig internet.ConnectionAuthenticator
	config     *Config
	addConn    internet.ConnHandler
	locker     *internet.FileLocker // for unix domain socket
}

// ListenTCP creates a new Listener based on configurations.
func ListenTCP(ctx context.Context, address net.Address, port net.Port, streamSettings *internet.MemoryStreamConfig, handler internet.ConnHandler) (internet.Listener, error) {
	fmt.Println("in transport-internet-tcp-hub.go func ListenTCP")
	l := &Listener{
		addConn: handler,
	}
	tcpSettings := streamSettings.ProtocolSettings.(*Config)
	l.config = tcpSettings
	if l.config != nil {
		fmt.Println("in transport-internet-tcp-hub.go func ListenTCP l.config != nil: ", l.config)
		if streamSettings.SocketSettings == nil {
			
			streamSettings.SocketSettings = &internet.SocketConfig{}
			//gztest.GetMessageReflectType(streamSettings.SocketSettings)
		}
		streamSettings.SocketSettings.AcceptProxyProtocol = l.config.AcceptProxyProtocol
		
	}
	var listener net.Listener
	var err error
	if port == net.Port(0) { // unix
		listener, err = internet.ListenSystem(ctx, &net.UnixAddr{
			Name: address.Domain(),
			Net:  "unix",
		}, streamSettings.SocketSettings)
		if err != nil {
			return nil, errors.New("failed to listen Unix Domain Socket on ")
		}
		fmt.Println("listening Unix Domain Socket on ", address)
		locker := ctx.Value(address.Domain())
		if locker != nil {
			l.locker = locker.(*internet.FileLocker)
		}
	} else {
		fmt.Println("in transport-internet-tcp-hub.go func ListenTCP ctx: ", ctx)
		//gztest.GetMessageReflectType(streamSettings.SocketSettings)
		listener, err = internet.ListenSystem(ctx, &net.TCPAddr{
			IP:   address.IP(),
			Port: int(port),
		}, streamSettings.SocketSettings)
		if err != nil {
			return nil, errors.New("failed to listen TCP on")
		}
		fmt.Println("listening TCP on ", address, ":", port)
	}

	if streamSettings.SocketSettings != nil && streamSettings.SocketSettings.AcceptProxyProtocol {
		fmt.Println("accepting PROXY protocol")
	}

	l.listener = listener

	if config := tls.ConfigFromStreamSettings(streamSettings); config != nil {
		l.tlsConfig = config.GetTLSConfig()
	}

	if tcpSettings.HeaderSettings != nil {
		headerConfig, err := tcpSettings.HeaderSettings.GetInstance()
		if err != nil {
			return nil, errors.New("invalid header settings")
		}
		auth, err := internet.CreateConnectionAuthenticator(headerConfig)
		if err != nil {
			return nil, errors.New("invalid header settings")
		}
		l.authConfig = auth
	}
// 开启一个协程，监听接收的数据
	go l.keepAccepting()
	//fmt.Println("测试 callback 顺序 in hub.go ListenTCP")
	return l, nil
}

func (v *Listener) keepAccepting() {
	fmt.Println("in transport-internet-tcp-hub.go func (v *Listener) keepAccepting()")
	for {
		fmt.Println("等待转入下一个 TCP 连接的信号")
		// Accept 等待并返回下一个连接给监听器
		conn, err := v.listener.Accept()		
		if err != nil {
			fmt.Println("测试是否在等待信号.......")
			errStr := err.Error()
			if strings.Contains(errStr, "closed") {
				break
			}
			fmt.Println("failed to accepted raw connections")
			if strings.Contains(errStr, "too many") {
				time.Sleep(time.Millisecond * 500)
			}
			continue
		}
// Config 结构用于配置 TLS 客户端或服务器。
		if v.tlsConfig != nil {
			fmt.Println("in transport-internet-tcp-hub.go func (v *Listener) keepAccepting() v.tlsConfig != nil ")
			conn = tls.Server(conn, v.tlsConfig)
		}
		if v.authConfig != nil {
			fmt.Println("in transport-internet-tcp-hub.go func (v *Listener) keepAccepting() v.authConfig != nil ")
			conn = v.authConfig.Server(conn)
		}
		fmt.Println("转入下一个 TCP 连接")
		v.addConn(internet.Connection(conn))
		//fmt.Println("测试 callback 顺序")
	}
	fmt.Println("in transport-internet-tcp-hub.go func (v *Listener) keepAccepting()  END")
}

// Addr implements internet.Listener.Addr.
func (v *Listener) Addr() net.Addr {
	return v.listener.Addr()
}

// Close implements internet.Listener.Close.
func (v *Listener) Close() error {
	fmt.Println("in transport-internet-tcp-hub.go func (v *Listener) Close()")
	if v.locker != nil {
		v.locker.Release()
	}
	return v.listener.Close()
}

func init() {
	fmt.Println("in transport-internet-tcp-hub.go func init()")
	common.Must(internet.RegisterTransportListener(protocolName, ListenTCP))
}
