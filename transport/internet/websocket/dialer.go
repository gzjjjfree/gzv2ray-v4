//go:build !confonly
// +build !confonly

package websocket

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"time"
	"fmt"

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/features/ext"

	"github.com/gorilla/websocket"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/session"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/tls"
)

// Dial dials a WebSocket connection to the given destination.
func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (internet.Connection, error) {
	fmt.Println("in transport-internet-websocket-dialer.go func Dial creating connection to: ", dest)
	newError("creating connection to ", dest).WriteToLog(session.ExportIDToError(ctx))

	conn, err := dialWebsocket(ctx, dest, streamSettings)
	if err != nil {
		return nil, newError("failed to dial WebSocket").Base(err)
	}
	return internet.Connection(conn), nil
}

func init() {
	fmt.Println("in transport-internet-websocket-clialer.go func init()")
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}

func dialWebsocket(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (net.Conn, error) {
	fmt.Println("in transport-internet-websocket-dialer.go func dialWebsocket")
	wsSettings := streamSettings.ProtocolSettings.(*Config)

	dialer := &websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return internet.DialSystem(ctx, dest, streamSettings.SocketSettings)
		},
		ReadBufferSize:   4 * 1024,
		WriteBufferSize:  4 * 1024,
		HandshakeTimeout: time.Second * 8,
	}

	protocol := "ws"

	if config := tls.ConfigFromStreamSettings(streamSettings); config != nil {
		protocol = "wss"
		dialer.TLSClientConfig = config.GetTLSConfig(tls.WithDestination(dest), tls.WithNextProto("http/1.1"))
	}

	host := dest.NetAddr()
	
	if (protocol == "ws" && dest.Port == 80) || (protocol == "wss" && dest.Port == 443) {
		host = dest.Address.String()
	}
	fmt.Println("in transport-internet-websocket-dialer.go func dialWebsocket host is: ", host)
	uri := protocol + "://" + host + wsSettings.GetNormalizedPath()

	if wsSettings.UseBrowserForwarding {
		var forwarder ext.BrowserForwarder
		err := core.RequireFeatures(ctx, func(Forwarder ext.BrowserForwarder) {
			forwarder = Forwarder
		})
		if err != nil {
			return nil, newError("cannot find browser forwarder service").Base(err)
		}
		if wsSettings.MaxEarlyData != 0 {
			return newRelayedConnectionWithDelayedDial(&dialerWithEarlyDataRelayed{
				forwarder: forwarder,
				uriBase:   uri,
				config:    wsSettings,
			}), nil
		}
		conn, err := forwarder.DialWebsocket(uri, nil)
		if err != nil {
			return nil, newError("cannot dial with browser forwarder service").Base(err)
		}
		return newRelayedConnection(conn), nil
	}

	if wsSettings.MaxEarlyData != 0 {
		return newConnectionWithDelayedDial(&dialerWithEarlyData{
			dialer:  dialer,
			uriBase: uri,
			config:  wsSettings,
		}), nil
	}
	// fmt.Println("in transport-internet-websocket-dialer.go func dialWebsocket dialer.Subprotocols is: ", dialer.Subprotocols)
	// 标准库函数 dialer.Dial 根据参数发起一个 ws 连接，握手成功后返回 conn 和 resp
	conn, resp, err := dialer.Dial(uri, wsSettings.GetRequestHeader())
	fmt.Println("in transport-internet-websocket-dialer.go func dialWebsocket resp is: ", resp)
	if err != nil {
		var reason string
		if resp != nil {
			reason = resp.Status
		}
		return nil, newError("failed to dial to (", uri, "): ", reason).Base(err)
	}

	return newConnection(conn, conn.RemoteAddr()), nil
}

type dialerWithEarlyData struct {
	dialer  *websocket.Dialer
	uriBase string
	config  *Config
}

func (d dialerWithEarlyData) Dial(earlyData []byte) (*websocket.Conn, error) {
	fmt.Println("in transport-internet-websocket-dialer.go func (d dialerWithEarlyData) Dial")
	earlyDataBuf := bytes.NewBuffer(nil)
	base64EarlyDataEncoder := base64.NewEncoder(base64.RawURLEncoding, earlyDataBuf)

	earlydata := bytes.NewReader(earlyData)
	limitedEarlyDatareader := io.LimitReader(earlydata, int64(d.config.MaxEarlyData))
	n, encerr := io.Copy(base64EarlyDataEncoder, limitedEarlyDatareader)
	if encerr != nil {
		return nil, newError("websocket delayed dialer cannot encode early data").Base(encerr)
	}

	if errc := base64EarlyDataEncoder.Close(); errc != nil {
		return nil, newError("websocket delayed dialer cannot encode early data tail").Base(errc)
	}

	conn, resp, err := d.dialer.Dial(d.uriBase+earlyDataBuf.String(), d.config.GetRequestHeader())
	if err != nil {
		var reason string
		if resp != nil {
			reason = resp.Status
		}
		return nil, newError("failed to dial to (", d.uriBase, ") with early data: ", reason).Base(err)
	}
	if n != int64(len(earlyData)) {
		if errWrite := conn.WriteMessage(websocket.BinaryMessage, earlyData[n:]); errWrite != nil {
			return nil, newError("failed to dial to (", d.uriBase, ") with early data as write of remainder early data failed: ").Base(err)
		}
	}
	return conn, nil
}

type dialerWithEarlyDataRelayed struct {
	forwarder ext.BrowserForwarder
	uriBase   string
	config    *Config
}

func (d dialerWithEarlyDataRelayed) Dial(earlyData []byte) (io.ReadWriteCloser, error) {
	fmt.Println("in transport-internet-websocket-dialer.go func (d dialerWithEarlyDataRelayed) Dial")
	earlyDataBuf := bytes.NewBuffer(nil)
	base64EarlyDataEncoder := base64.NewEncoder(base64.RawURLEncoding, earlyDataBuf)

	earlydata := bytes.NewReader(earlyData)
	limitedEarlyDatareader := io.LimitReader(earlydata, int64(d.config.MaxEarlyData))
	n, encerr := io.Copy(base64EarlyDataEncoder, limitedEarlyDatareader)
	if encerr != nil {
		return nil, newError("websocket delayed dialer cannot encode early data").Base(encerr)
	}

	if errc := base64EarlyDataEncoder.Close(); errc != nil {
		return nil, newError("websocket delayed dialer cannot encode early data tail").Base(errc)
	}

	conn, err := d.forwarder.DialWebsocket(d.uriBase+earlyDataBuf.String(), d.config.GetRequestHeader())
	if err != nil {
		var reason string
		return nil, newError("failed to dial to (", d.uriBase, ") with early data: ", reason).Base(err)
	}
	if n != int64(len(earlyData)) {
		if _, errWrite := conn.Write(earlyData[n:]); errWrite != nil {
			return nil, newError("failed to dial to (", d.uriBase, ") with early data as write of remainder early data failed: ").Base(err)
		}
	}
	return conn, nil
}
