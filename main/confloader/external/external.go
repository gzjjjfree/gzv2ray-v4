package external

import (
	"bytes"
	//"fmt"
	"io"
	//"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	//"example.com/gztest"

	"github.com/gzjjjfree/gzv2ray-v4/common/buf"
	"github.com/gzjjjfree/gzv2ray-v4/common/platform/ctlcmd"
	"github.com/gzjjjfree/gzv2ray-v4/main/confloader"
)

func ConfigLoader(arg string) (out io.Reader, err error) {
	var data []byte
	switch {
	case strings.HasPrefix(arg, "http://"), strings.HasPrefix(arg, "https://"):
		data, err = FetchHTTPContent(arg)

	case arg == "stdin:":
		data, err = io.ReadAll(os.Stdin)

	default:
		data, err = os.ReadFile(arg)
	}

	if err != nil {
		return
	}
	//fmt.Println("Print in main-confloader-extemal.go func Configloader data: ")
	//这是读入的config内容
	//fmt.Println(string(data))
	//gztest.GetMessageReflectType(data)
	out = bytes.NewBuffer(data)
	return
}

func FetchHTTPContent(target string) ([]byte, error) {
	parsedTarget, err := url.Parse(target)
	if err != nil {
		return nil, newError("invalid URL: ", target).Base(err)
	}

	if s := strings.ToLower(parsedTarget.Scheme); s != "http" && s != "https" {
		return nil, newError("invalid scheme: ", parsedTarget.Scheme)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(&http.Request{
		Method: "GET",
		URL:    parsedTarget,
		Close:  true,
	})
	if err != nil {
		return nil, newError("failed to dial to ", target).Base(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, newError("unexpected HTTP status code: ", resp.StatusCode)
	}

	content, err := buf.ReadAllToBytes(resp.Body)
	if err != nil {
		return nil, newError("failed to read HTTP response").Base(err)
	}

	return content, nil
}

func ExtConfigLoader(files []string, reader io.Reader) (io.Reader, error) {
	buf, err := ctlcmd.Run(append([]string{"config"}, files...), reader)
	if err != nil {
		return nil, err
	}

	return strings.NewReader(buf.String()), nil
}

func init() {
	confloader.EffectiveConfigFileLoader = ConfigLoader
	confloader.EffectiveExtConfigLoader = ExtConfigLoader
}
