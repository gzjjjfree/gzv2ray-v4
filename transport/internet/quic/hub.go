//go:build !confonly
// +build !confonly

package quic

import (
	"context"
	"time"
	"fmt"

	"github.com/quic-go/quic-go"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/protocol/tls/cert"
	"github.com/gzjjjfree/gzv2ray-v4/common/signal/done"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/tls"
)

// Listener is an internet.Listener that listens for TCP connections.
type Listener struct {
	rawConn  *sysConn
	listener *quic.Listener
	done     *done.Instance
	addConn  internet.ConnHandler
}

func (l *Listener) acceptStreams(session quic.Connection) {
	fmt.Println("in transport-internet-quic-hub.go func (l *Listener) acceptStreams")
	for {
		stream, err := session.AcceptStream(context.Background())
		if err != nil {
			newError("failed to accept stream").Base(err).WriteToLog()
			select {
			case <-session.Context().Done():
				return
			case <-l.done.Wait():
				if err := session.CloseWithError(0, ""); err != nil {
					newError("failed to close session").Base(err).WriteToLog()
				}
				return
			default:
				time.Sleep(time.Second)
				continue
			}
		}

		conn := &interConn{
			stream: stream,
			local:  session.LocalAddr(),
			remote: session.RemoteAddr(),
		}

		l.addConn(conn)
	}
}

func (l *Listener) keepAccepting() {
	fmt.Println("in transport-internet-quic-hub.go")
	for {
		conn, err := l.listener.Accept(context.Background())
		if err != nil {
			newError("failed to accept QUIC sessions").Base(err).WriteToLog()
			if l.done.Done() {
				break
			}
			time.Sleep(time.Second)
			continue
		}
		go l.acceptStreams(conn)
	}
}

// Addr implements internet.Listener.Addr.
func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}

// Close implements internet.Listener.Close.
func (l *Listener) Close() error {
	fmt.Println("in transport-internet-quic-hub.go func (l *Listener) Close()")
	l.done.Close()
	l.listener.Close()
	l.rawConn.Close()
	return nil
}

// Listen creates a new Listener based on configurations.
func Listen(ctx context.Context, address net.Address, port net.Port, streamSettings *internet.MemoryStreamConfig, handler internet.ConnHandler) (internet.Listener, error) {
	fmt.Println("in transport-internet-quic-hub.go func Listen")
	if address.Family().IsDomain() {
		return nil, newError("domain address is not allows for listening quic")
	}

	tlsConfig := tls.ConfigFromStreamSettings(streamSettings)
	if tlsConfig == nil {
		tlsConfig = &tls.Config{
			Certificate: []*tls.Certificate{tls.ParseCertificate(cert.MustGenerate(nil, cert.DNSNames(internalDomain), cert.CommonName(internalDomain)))},
		}
	}

	config := streamSettings.ProtocolSettings.(*Config)
	rawConn, err := internet.ListenSystemPacket(context.Background(), &net.UDPAddr{
		IP:   address.IP(),
		Port: int(port),
	}, streamSettings.SocketSettings)

	if err != nil {
		return nil, err
	}

	quicConfig := &quic.Config{
		HandshakeIdleTimeout:  time.Second * 8,
		MaxIdleTimeout:        time.Second * 45,
		MaxIncomingStreams:    32,
		MaxIncomingUniStreams: -1,
		KeepAlivePeriod:       time.Second * 15,
	}

	conn, err := wrapSysConn(rawConn.(*net.UDPConn), config)
	if err != nil {
		conn.Close()
		return nil, err
	}

	tr := quic.Transport{
		Conn:               conn,
		ConnectionIDLength: 12,
	}

	qListener, err := tr.Listen(tlsConfig.GetTLSConfig(), quicConfig)
	if err != nil {
		conn.Close()
		return nil, err
	}

	listener := &Listener{
		done:     done.New(),
		rawConn:  conn,
		listener: qListener,
		addConn:  handler,
	}

	go listener.keepAccepting()

	return listener, nil
}

func init() {
	common.Must(internet.RegisterTransportListener(protocolName, Listen))
}
