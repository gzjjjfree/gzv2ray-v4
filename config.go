//go:build !confonly
// +build !confonly

package core

import (
	"errors"
	"fmt"
	"io"
	"strings"

	//"example.com/gztest"

	"google.golang.org/protobuf/proto"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/buf"
	"github.com/gzjjjfree/gzv2ray-v4/common/cmdarg"
	"github.com/gzjjjfree/gzv2ray-v4/main/confloader"
)

// LoadConfig loads config with given format from given source.
// LoadConfig 从给定源加载具有给定格式的配置
// input accepts 2 different types:
// 输入接受两种不同的类型：
// * []string slice of multiple filename/url(s) to open to read
// * []string 切片，包含多个文件名/url，用于打开并读取
// * io.Reader that reads a config content (the original way)
// io.Reader 读取配置内容（原始方式）
func LoadConfig(formatName string, filename string, input interface{}) (*Config, error) { // formatName 文件的类型，filename 文件名切片，input 通过 io.Reader 读取
	ext := getExtension(filename) // ext 找 filename 后缀名
	//fmt.Println("formatName is: ", formatName)
	//fmt.Println("filename is: ", filename)
	//for key := range configLoaderByExt {
	//	fmt.Printf("key=%s ", key)
	//}
	//fmt.Println("configLoaderByExt is: ", configLoaderByExt)
	//fmt.Println("configLoaderByExt is: ", configLoaderByExt)
	if len(ext) > 0 {
		if f, found := configLoaderByExt[ext]; found { // 如果是 v2ray 的可配置格式，初始是pb
			//fmt.Println("configLoaderByExt[ext] is right")
			//gztest.GetMessageReflectType(*f)
			return f.Loader(input) // 返回配置加载函数，把文件加载到接口中
		}
	}
	//fmt.Println("configLoaderByName is: ", configLoaderByName)
	if f, found := configLoaderByName[formatName]; found { // 通过 formatName 类型确认格式
		return f.Loader(input)
	}

	return nil, fmt.Errorf("unable to load config in %v", formatName)
}

func getExtension(filename string) string {
	idx := strings.LastIndexByte(filename, '.') // LastIndexByte 返回 s 中 c 的最后一个实例的索引，如果 c 不存在于 s 中，则返回 -1，这里读取路径中的文件名后缀前的符号 .
	if idx == -1 {                              // 如果没有，返回空
		return ""
	}
	return filename[idx+1:] // . 后面是后缀名
}

var (
	configLoaderByName = make(map[string]*ConfigFormat)
	configLoaderByExt  = make(map[string]*ConfigFormat)
)

// ConfigFormat is a configurable format of V2Ray config file.
type ConfigFormat struct { // ConfigFormat 是 V2Ray 配置文件的可配置格式。
	Name      string
	Extension []string
	Loader    ConfigLoader
}

// ConfigLoader is a utility to load V2Ray config from external source.
type ConfigLoader func(input interface{}) (*Config, error) // ConfigLoader 是一个从外部源加载 V2Ray 配置的实用程序。

// RegisterConfigLoader add a new ConfigLoader.
// RegisterConfigLoader 添加一个新的 ConfigLoader。
func RegisterConfigLoader(format *ConfigFormat) error {
	name := strings.ToLower(format.Name)
	if _, found := configLoaderByName[name]; found {
		return errors.New(" already registered")
	}
	configLoaderByName[name] = format

	for _, ext := range format.Extension {
		lext := strings.ToLower(ext)
		if _, found := configLoaderByExt[lext]; found {
			return errors.New(" already registered to ")
		}
		configLoaderByExt[lext] = format
	}

	return nil
}

func loadProtobufConfig(data []byte) (*Config, error) {
	fmt.Println("in ./config.go func loadProtobufConfig data is:")
	fmt.Println(string(data))
	config := new(Config)
	if err := proto.Unmarshal(data, config); err != nil {
		return nil, err
	}
	//fmt.Println("in config.go Print the *config.app:")
	//est.GetMessageReflectType(config.App)
	return config, nil
}

func init() {
	fmt.Println("in is run ./config.go func init ")
	common.Must(RegisterConfigLoader(&ConfigFormat{
		Name:      "Protobuf",
		Extension: []string{"pb"},
		Loader: func(input interface{}) (*Config, error) {
			switch v := input.(type) {
			case cmdarg.Arg:
				r, err := confloader.LoadConfig(v[0])
				common.Must(err)
				data, err := buf.ReadAllToBytes(r)
				common.Must(err)
				return loadProtobufConfig(data)
			case io.Reader:
				data, err := buf.ReadAllToBytes(v)
				common.Must(err)
				return loadProtobufConfig(data)
			default:
				//fmt.Println("unknow type")
				return nil, errors.New("unknow type")
			}
		},
	}))
}
