// +build !confonly

package browserforwarder

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
	"errors"
	"fmt"

	

	"github.com/gzjjjfree/gzv2ray-v4/app/browserforwarder/handler"

	//"github.com/v2fly/BrowserBridge/handler"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/platform/securedload"
	"github.com/gzjjjfree/gzv2ray-v4/features/ext"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
)


type Forwarder struct {
	ctx context.Context

	forwarder  *handler.HTTPHandle
	httpserver *http.Server

	config *Config
}

func (f *Forwarder) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	requestPath := request.URL.Path[1:]

	switch requestPath {
	case "":
		fallthrough
	case "index.js":
		BridgeResource(writer, request, requestPath)
	case "link":
		f.forwarder.ServeBridge(writer, request)
	}
}

func (f *Forwarder) DialWebsocket(url string, header http.Header) (io.ReadWriteCloser, error) {
	return f.forwarder.Dial(url)
}

func (f *Forwarder) Type() interface{} {
	return ext.BrowserForwarderType()
}

func (f *Forwarder) Start() error {
	fmt.Println("in app-browserforwarder-forwarder.go func Start()")
	f.forwarder = handler.NewHttpHandle()
	f.httpserver = &http.Server{Handler: f}
	address := net.ParseAddress(f.config.ListenAddr)
	listener, err := internet.ListenSystem(f.ctx, &net.TCPAddr{IP: address.IP(), Port: int(f.config.ListenPort)}, nil)
	if err != nil {
		return errors.New("forwarder cannot listen on the port")
	}
	go func() {
		err = f.httpserver.Serve(listener)
		if err != nil {
			fmt.Println("cannot serve http forward server")
		}
	}()
	return nil
}

func (f *Forwarder) Close() error {
	fmt.Println("in app-browserforwarder-forwarder.go func (f *Forwarder) Close()")
	if f.httpserver != nil {
		return f.httpserver.Close()
	}
	return nil
}

func BridgeResource(rw http.ResponseWriter, r *http.Request, path string) {
	content := path
	if content == "" {
		content = "index.html"
	}
	data, err := securedload.GetAssetSecured("browserforwarder/" + content)
	if err != nil {
		err = errors.New("cannot load necessary resources")
		http.Error(rw, err.Error(), http.StatusForbidden)
		return
	}

	http.ServeContent(rw, r, path, time.Now(), bytes.NewReader(data))
}

func NewForwarder(ctx context.Context, cfg *Config) *Forwarder {
	return &Forwarder{config: cfg, ctx: ctx}
}

func init() {
	fmt.Println("is run ./app/browserforwarder/forwarder.go func init ")
	common.Must(common.RegisterConfig((*Config)(nil), func(ctx context.Context, cfg interface{}) (interface{}, error) {
		return NewForwarder(ctx, cfg.(*Config)), nil
	}))
}
