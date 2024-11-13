package json

import (
	"errors"
	"fmt"
	"io"
	"os"

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/cmdarg"
	"github.com/gzjjjfree/gzv2ray-v4/main/confloader"
)

func init() {
	fmt.Println("run main-json func init")
	common.Must(core.RegisterConfigLoader(&core.ConfigFormat{
		Name:      "JSON",
		Extension: []string{"json"},
		Loader: func(input interface{}) (*core.Config, error) {
			switch v := input.(type) {
			case cmdarg.Arg:
				r, err := confloader.LoadExtConfig(v, os.Stdin)
				if err != nil {
					return nil, errors.New("failed to execute v2ctl to convert config file")
				}
				return core.LoadConfig("protobuf", "", r)
			case io.Reader:
				r, err := confloader.LoadExtConfig([]string{"stdin:"}, os.Stdin)
				if err != nil {
					return nil, errors.New("failed to execute v2ctl to convert config file")
				}
				return core.LoadConfig("protobuf", "", r)
			default:
				return nil, errors.New("unknown type")
			}
		},
	}))
}
