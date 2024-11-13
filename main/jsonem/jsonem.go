package jsonem

import (
	"errors"
	"fmt"
	"io"

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/cmdarg"
	"github.com/gzjjjfree/gzv2ray-v4/infra/conf"
	"github.com/gzjjjfree/gzv2ray-v4/infra/conf/serial"
	"github.com/gzjjjfree/gzv2ray-v4/main/confloader"
)

func init() {
	fmt.Println("run main-jsonem func init")
	common.Must(core.RegisterConfigLoader(&core.ConfigFormat{
		Name:      "JSON",
		Extension: []string{"json"},
		Loader: func(input interface{}) (*core.Config, error) {
			switch v := input.(type) {
			case cmdarg.Arg:
				cf := &conf.Config{}
				for i, arg := range v {
					fmt.Println("Reading config: ", arg)
					r, err := confloader.LoadConfig(arg)
					common.Must(err)
					c, err := serial.DecodeJSONConfig(r)
					common.Must(err)
					if i == 0 {
						// This ensure even if the muti-json parser do not support a setting,
						// It is still respected automatically for the first configure file
						*cf = *c
						continue
					}
					cf.Override(c, arg)
				}
				// config 设置了 APP 及 inbounds 和 outbounds 字段
				return cf.Build()
			case io.Reader:
				return serial.LoadJSONConfig(v)
			default:
				return nil, errors.New("unknow type")
			}
		},
	}))
}
