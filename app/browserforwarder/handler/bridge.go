package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gzjjjfree/gzv2ray-v4/app/browserforwarder/handler/websocket"
	"github.com/gzjjjfree/gzv2ray-v4/app/browserforwarder/handler/smux"

	"github.com/gzjjjfree/gzv2ray-v4/app/browserforwarder/handler/websocket/websocketadp"
)

func (hs HTTPHandle) ServeBridge(rw http.ResponseWriter, r *http.Request) {
	if hs.link.bridgemux != nil {
		return
	}
	log.Println("reflective server connected")

	upg := websocket.Upgrader{}
	conn, err := upg.Upgrade(rw, r, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	wsconn := websocketadp.NewWsAdp(conn)

	hs.link.bridgemux, err = smux.Server(wsconn, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for {
		stream, err := hs.link.bridgemux.Accept()
		if err != nil {
			fmt.Println(err.Error())
			hs.link.bridgemux = nil
			return
		}
		go func() {
			io.Copy(os.Stdout, stream)
			stream.Close()
		}()

	}
}
