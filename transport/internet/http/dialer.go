//go:build !confonly
// +build !confonly

package http

import (
	"context"
	gotls "crypto/tls"
	"net/http"
	"net/url"
	"sync"
	"fmt"

	"golang.org/x/net/http2"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/buf"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet/tls"
	"github.com/gzjjjfree/gzv2ray-v4/transport/pipe"
)

var (
	globalDialerMap    map[net.Destination]*http.Client
	globalDialerAccess sync.Mutex
)

func getHTTPClient(_ context.Context, dest net.Destination, tlsSettings *tls.Config) *http.Client {
	fmt.Println("in transport-internet-http-clialer.go func getHTTPClient")
	globalDialerAccess.Lock()
	defer globalDialerAccess.Unlock()

	if globalDialerMap == nil {
		globalDialerMap = make(map[net.Destination]*http.Client)
	}

	if client, found := globalDialerMap[dest]; found {
		return client
	}

	transport := &http2.Transport{
		DialTLS: func(network string, addr string, tlsConfig *gotls.Config) (net.Conn, error) {
			rawHost, rawPort, err := net.SplitHostPort(addr)
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

			pconn, err := internet.DialSystem(context.Background(), net.TCPDestination(address, port), nil)
			if err != nil {
				return nil, err
			}

			cn := gotls.Client(pconn, tlsConfig)
			if err := cn.Handshake(); err != nil {
				return nil, err
			}
			if !tlsConfig.InsecureSkipVerify {
				if err := cn.VerifyHostname(tlsConfig.ServerName); err != nil {
					return nil, err
				}
			}
			state := cn.ConnectionState()
			if p := state.NegotiatedProtocol; p != http2.NextProtoTLS {
				return nil, newError("http2: unexpected ALPN protocol " + p + "; want q" + http2.NextProtoTLS).AtError()
			}
			return cn, nil
		},
		TLSClientConfig: tlsSettings.GetTLSConfig(tls.WithDestination(dest)),
	}

	client := &http.Client{
		Transport: transport,
	}

	globalDialerMap[dest] = client
	return client
}

// Dial dials a new TCP connection to the given destination.
func Dial(ctx context.Context, dest net.Destination, streamSettings *internet.MemoryStreamConfig) (internet.Connection, error) {
	fmt.Println("in transport-internet-http-clialer.go func Dial dest is: ", dest)
	httpSettings := streamSettings.ProtocolSettings.(*Config)
	tlsConfig := tls.ConfigFromStreamSettings(streamSettings)
	if tlsConfig == nil {
		return nil, newError("TLS must be enabled for http transport.").AtWarning()
	}
	client := getHTTPClient(ctx, dest, tlsConfig)

	opts := pipe.OptionsFromContext(ctx)
	preader, pwriter := pipe.New(opts...)
	breader := &buf.BufferedReader{Reader: preader}
	request := &http.Request{
		Method: "PUT",
		Host:   httpSettings.getRandomHost(),
		Body:   breader,
		URL: &url.URL{
			Scheme: "https",
			Host:   dest.NetAddr(),
			Path:   httpSettings.getNormalizedPath(),
		},
		Proto:      "HTTP/2",
		ProtoMajor: 2,
		ProtoMinor: 0,
		Header:     make(http.Header),
	}
	// Disable any compression method from server.
	request.Header.Set("Accept-Encoding", "identity")

	response, err := client.Do(request) // nolint: bodyclose
	if err != nil {
		return nil, newError("failed to dial to ", dest).Base(err).AtWarning()
	}
	if response.StatusCode != 200 {
		return nil, newError("unexpected status", response.StatusCode).AtWarning()
	}

	bwriter := buf.NewBufferedWriter(pwriter)
	common.Must(bwriter.SetBuffered(false))
	return net.NewConnection(
		net.ConnectionOutput(response.Body),
		net.ConnectionInput(bwriter),
		net.ConnectionOnClose(common.ChainedClosable{breader, bwriter, response.Body}),
	), nil
}

func init() {
	fmt.Println("in transport-internet-http-clialer.go func init()")
	common.Must(internet.RegisterTransportDialer(protocolName, Dial))
}
