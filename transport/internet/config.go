package internet

import (
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/features"
	"github.com/gzjjjfree/gzv2ray-v4/common/serial"
)

type ConfigCreator func() interface{}

var (
	globalTransportConfigCreatorCache = make(map[string]ConfigCreator)
	globalTransportSettings           []*TransportConfig
)

const unknownProtocol = "unknown"

func transportProtocolToString(protocol TransportProtocol) string {
	//fmt.Println("in transport-internet-config.go func transportProtocolToString(protocol TransportProtocol)")
	switch protocol {
	case TransportProtocol_TCP:
		return "tcp"
	case TransportProtocol_UDP:
		return "udp"
	case TransportProtocol_HTTP:
		return "http"
	case TransportProtocol_MKCP:
		return "mkcp"
	case TransportProtocol_WebSocket:
		return "websocket"
	case TransportProtocol_DomainSocket:
		return "domainsocket"
	default:
		return unknownProtocol
	}
}

func RegisterProtocolConfigCreator(name string, creator ConfigCreator) error {
	fmt.Println("in transport-internet-config.go func RegisterProtocolConfigCreator(name string, creator ConfigCreator): ", name)
	if _, found := globalTransportConfigCreatorCache[name]; found {
		return errors.New("protocol is already registered")
	}
	globalTransportConfigCreatorCache[name] = creator
	return nil
}

func CreateTransportConfig(name string) (interface{}, error) {
	//fmt.Println("in transport-internet-config.go func CreateTransportConfig(name string)")
	creator, ok := globalTransportConfigCreatorCache[name]
	if !ok {
		return nil, errors.New("unknown transport protocol: ")
	}
	return creator(), nil
}

func (c *TransportConfig) GetTypedSettings() (interface{}, error) {
	//fmt.Println("in transport-internet-config.go func (c *TransportConfig) GetTypedSettings()")
	return c.Settings.GetInstance()
}

func (c *TransportConfig) GetUnifiedProtocolName() string {
	//fmt.Println("in transport-internet-config.go func (c *TransportConfig) GetUnifiedProtocolName()")
	if len(c.ProtocolName) > 0 {
		return c.ProtocolName
	}

	return transportProtocolToString(c.Protocol)
}

func (c *StreamConfig) GetEffectiveProtocol() string {
	//fmt.Println("in transport-internet-config.go func (c *StreamConfig) GetEffectiveProtocol()")
	if c == nil {
		return "tcp"
	}

	if len(c.ProtocolName) > 0 {
		return c.ProtocolName
	}

	return transportProtocolToString(c.Protocol)
}

func (c *StreamConfig) GetEffectiveTransportSettings() (interface{}, error) {
	//fmt.Println("in transport-internet-config.go func (c *StreamConfig) GetEffectiveTransportSettings()")
	protocol := c.GetEffectiveProtocol()
	return c.GetTransportSettingsFor(protocol)
}

func (c *StreamConfig) GetTransportSettingsFor(protocol string) (interface{}, error) {
	//fmt.Println("in transport-internet-config.go func (c *StreamConfig) GetTransportSettingsFor(protocol string)")
	if c != nil {
		for _, settings := range c.TransportSettings {
			if settings.GetUnifiedProtocolName() == protocol {
				return settings.GetTypedSettings()
			}
		}
	}

	for _, settings := range globalTransportSettings {
		if settings.GetUnifiedProtocolName() == protocol {
			return settings.GetTypedSettings()
		}
	}

	return CreateTransportConfig(protocol)
}

func (c *StreamConfig) GetEffectiveSecuritySettings() (interface{}, error) {
	//fmt.Println("in transport-internet-config.go func (c *StreamConfig) GetEffectiveSecuritySettings()")
	for _, settings := range c.SecuritySettings {
		if settings.Type == c.SecurityType {
			return settings.GetInstance()
		}
	}
	return serial.GetInstance(c.SecurityType)
}

func (c *StreamConfig) HasSecuritySettings() bool {
	//fmt.Println("in transport-internet-config.go func (c *StreamConfig) HasSecuritySettings()")
	return len(c.SecurityType) > 0
}

func ApplyGlobalTransportSettings(settings []*TransportConfig) error {
	//fmt.Println("in transport-internet-config.go func ApplyGlobalTransportSettings(settings []*TransportConfig)")
	features.PrintDeprecatedFeatureWarning("global transport settings")
	globalTransportSettings = settings
	return nil
}

func (c *ProxyConfig) HasTag() bool {
	return c != nil && len(c.Tag) > 0
}

func (m SocketConfig_TProxyMode) IsEnabled() bool {
	return m != SocketConfig_Off
}
