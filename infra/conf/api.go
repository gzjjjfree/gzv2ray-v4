package conf

import (
	"strings"
	"errors"

	"github.com/gzjjjfree/gzv2ray-v4/app/commander"
	loggerservice "github.com/gzjjjfree/gzv2ray-v4/app/log/command"
	handlerservice "github.com/gzjjjfree/gzv2ray-v4/app/proxyman/command"
	statsservice "github.com/gzjjjfree/gzv2ray-v4/app/stats/command"
	"github.com/gzjjjfree/gzv2ray-v4/common/serial"
)

type APIConfig struct {
	Tag      string   `json:"tag"`
	Services []string `json:"services"`
}

func (c *APIConfig) Build() (*commander.Config, error) {
	if c.Tag == "" {
		return nil, errors.New("aPI tag can't be empty")
	}

	services := make([]*serial.TypedMessage, 0, 16)
	for _, s := range c.Services {
		switch strings.ToLower(s) {
		case "reflectionservice":
			services = append(services, serial.ToTypedMessage(&commander.ReflectionConfig{}))
		case "handlerservice":
			services = append(services, serial.ToTypedMessage(&handlerservice.Config{}))
		case "loggerservice":
			services = append(services, serial.ToTypedMessage(&loggerservice.Config{}))
		case "statsservice":
			services = append(services, serial.ToTypedMessage(&statsservice.Config{}))
		}
	}

	return &commander.Config{
		Tag:     c.Tag,
		Service: services,
	}, nil
}
