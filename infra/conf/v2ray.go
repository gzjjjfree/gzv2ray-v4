package conf

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	//"reflect"
	"strings"
	//"path/filepath"

	//"reflect"

	//"example.com/gztest" //gztest.GetMessageReflectType(appSettings)

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/app/dispatcher"
	"github.com/gzjjjfree/gzv2ray-v4/app/proxyman"
	"github.com/gzjjjfree/gzv2ray-v4/app/stats"
	//"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/serial"
)

var (
	inboundConfigLoader = NewJSONConfigLoader(ConfigCreatorCache{
		"dokodemo-door": func() interface{} { return new(DokodemoConfig) },
		"http":          func() interface{} { return new(HTTPServerConfig) },
		"shadowsocks":   func() interface{} { return new(ShadowsocksServerConfig) },
		"socks":         func() interface{} { return new(SocksServerConfig) },
		//"vless":         func() interface{} { return new(VLessInboundConfig) },
		"vmess":   func() interface{} { return new(VMessInboundConfig) },
		"trojan":  func() interface{} { return new(TrojanServerConfig) },
		"mtproto": func() interface{} { return new(MTProtoServerConfig) },
	}, "protocol", "settings")

	outboundConfigLoader = NewJSONConfigLoader(ConfigCreatorCache{
		"blackhole":   func() interface{} { return new(BlackholeConfig) },
		"freedom":     func() interface{} { return new(FreedomConfig) },
		"http":        func() interface{} { return new(HTTPClientConfig) },
		"shadowsocks": func() interface{} { return new(ShadowsocksClientConfig) },
		"socks":       func() interface{} { return new(SocksClientConfig) },
		//"vless":       func() interface{} { return new(VLessOutboundConfig) },
		"vmess":    func() interface{} { return new(VMessOutboundConfig) },
		"trojan":   func() interface{} { return new(TrojanClientConfig) },
		"mtproto":  func() interface{} { return new(MTProtoClientConfig) },
		"dns":      func() interface{} { return new(DNSOutboundConfig) },
		"loopback": func() interface{} { return new(LoopbackConfig) },
	}, "protocol", "settings")

	ctllog = log.New(os.Stderr, "v2ctl> ", 0)
)

func toProtocolList(s []string) ([]proxyman.KnownProtocols, error) {
	kp := make([]proxyman.KnownProtocols, 0, 8)
	for _, p := range s {
		switch strings.ToLower(p) {
		case "http":
			kp = append(kp, proxyman.KnownProtocols_HTTP)
		case "https", "tls", "ssl":
			kp = append(kp, proxyman.KnownProtocols_TLS)
		default:
			return nil, errors.New("unknown protocol")
		}
	}
	return kp, nil
}

type SniffingConfig struct {
	Enabled      bool        `json:"enabled"`
	DestOverride *StringList `json:"destOverride"`
	MetadataOnly bool        `json:"metadataOnly"`
}

// Build implements Buildable.
func (c *SniffingConfig) Build() (*proxyman.SniffingConfig, error) {
	var p []string
	if c.DestOverride != nil {
		for _, domainOverride := range *c.DestOverride {
			switch strings.ToLower(domainOverride) {
			case "http":
				p = append(p, "http")
			case "tls", "https", "ssl":
				p = append(p, "tls")
			case "fakedns":
				p = append(p, "fakedns")
			default:
				return nil, errors.New("unknown protocol")
			}
		}
	}

	return &proxyman.SniffingConfig{
		Enabled:             c.Enabled,
		DestinationOverride: p,
		MetadataOnly:        c.MetadataOnly,
	}, nil
}

type MuxConfig struct {
	Enabled     bool  `json:"enabled"`
	Concurrency int16 `json:"concurrency"`
}

// Build creates MultiplexingConfig, Concurrency < 0 completely disables mux.
func (m *MuxConfig) Build() *proxyman.MultiplexingConfig {
	if m.Concurrency < 0 {
		return nil
	}

	var con uint32 = 8
	if m.Concurrency > 0 {
		con = uint32(m.Concurrency)
	}

	return &proxyman.MultiplexingConfig{
		Enabled:     m.Enabled,
		Concurrency: con,
	}
}

type InboundDetourAllocationConfig struct {
	Strategy    string  `json:"strategy"`
	Concurrency *uint32 `json:"concurrency"`
	RefreshMin  *uint32 `json:"refresh"`
}

// Build implements Buildable.
func (c *InboundDetourAllocationConfig) Build() (*proxyman.AllocationStrategy, error) {
	config := new(proxyman.AllocationStrategy)
	switch strings.ToLower(c.Strategy) {
	case "always":
		config.Type = proxyman.AllocationStrategy_Always
	case "random":
		config.Type = proxyman.AllocationStrategy_Random
	case "external":
		config.Type = proxyman.AllocationStrategy_External
	default:
		return nil, errors.New("unknown allocation strategy")
	}
	if c.Concurrency != nil {
		config.Concurrency = &proxyman.AllocationStrategy_AllocationStrategyConcurrency{
			Value: *c.Concurrency,
		}
	}

	if c.RefreshMin != nil {
		config.Refresh = &proxyman.AllocationStrategy_AllocationStrategyRefresh{
			Value: *c.RefreshMin,
		}
	}

	return config, nil
}

type InboundDetourConfig struct {
	Protocol       string                         `json:"protocol"`
	PortRange      *PortRange                     `json:"port"`
	ListenOn       *Address                       `json:"listen"`
	Settings       *json.RawMessage               `json:"settings"`
	Tag            string                         `json:"tag"`
	Allocation     *InboundDetourAllocationConfig `json:"allocate"`
	StreamSetting  *StreamConfig                  `json:"streamSettings"`
	DomainOverride *StringList                    `json:"domainOverride"`
	SniffingConfig *SniffingConfig                `json:"sniffing"`
}

// Build implements Buildable. 构建实现可构建
func (c *InboundDetourConfig) Build() (*core.InboundHandlerConfig, error) {
	receiverSettings := &proxyman.ReceiverConfig{}

	// 入站没有监听地址
	if c.ListenOn == nil {
		// Listen on anyip, must set PortRange 连端口都没有，返回错误
		if c.PortRange == nil {
			return nil, errors.New(";isten on AnyIP but no Port(s) set in InboundDetour")
		}
		// 端口变为 from and to
		receiverSettings.PortRange = c.PortRange.Build()
	} else {
		// Listen on specific IP or Unix Domain Socket
		// 监听特定 IP 或 Unix 域套接字
		receiverSettings.Listen = c.ListenOn.Build()
		listenDS := c.ListenOn.Family().IsDomain() && (c.ListenOn.Domain()[0] == '/' || c.ListenOn.Domain()[0] == '@')
		listenIP := c.ListenOn.Family().IsIP() || (c.ListenOn.Family().IsDomain() && c.ListenOn.Domain() == "localhost")
		switch {
		case listenIP:
			// Listen on specific IP, must set PortRange 监听特定IP，必须设置PortRange
			if c.PortRange == nil {
				return nil, errors.New("listen on specific ip without port in InboundDetour")
			}
			// Listen on IP:Port
			receiverSettings.PortRange = c.PortRange.Build()
		case listenDS:
			if c.PortRange != nil {
				// Listen on Unix Domain Socket, PortRange should be nil
				// 监听 Unix 域套接字，PortRange 应为零
				receiverSettings.PortRange = nil
			}
		default:
			return nil, errors.New("unable to listen on domain address")
		}
	}

	if c.Allocation != nil {
		concurrency := -1
		if c.Allocation.Concurrency != nil && c.Allocation.Strategy == "random" {
			concurrency = int(*c.Allocation.Concurrency)
		}
		portRange := int(c.PortRange.To - c.PortRange.From + 1)
		if concurrency >= 0 && concurrency >= portRange {
			return nil, errors.New("not enough ports. concurrency")
		}

		as, err := c.Allocation.Build()
		if err != nil {
			return nil, err
		}
		receiverSettings.AllocationStrategy = as
	}
	if c.StreamSetting != nil {
		ss, err := c.StreamSetting.Build()
		if err != nil {
			return nil, err
		}
		receiverSettings.StreamSettings = ss
	}
	if c.SniffingConfig != nil {
		s, err := c.SniffingConfig.Build()
		if err != nil {
			return nil, errors.New("failed to build sniffing config")
		}
		receiverSettings.SniffingSettings = s
	}
	if c.DomainOverride != nil {
		kp, err := toProtocolList(*c.DomainOverride)
		if err != nil {
			return nil, errors.New("failed to parse inbound detour config")
		}
		receiverSettings.DomainOverride = kp
	}

	settings := []byte("{}")
	if c.Settings != nil {
		settings = ([]byte)(*c.Settings)
	}
	rawConfig, err := inboundConfigLoader.LoadWithID(settings, c.Protocol)
	if err != nil {
		return nil, errors.New("failed to load inbound detour config")
	}
	if dokodemoConfig, ok := rawConfig.(*DokodemoConfig); ok {
		receiverSettings.ReceiveOriginalDestination = dokodemoConfig.Redirect
	}
	ts, err := rawConfig.(Buildable).Build()
	if err != nil {
		return nil, err
	}

	return &core.InboundHandlerConfig{
		Tag:              c.Tag,
		ReceiverSettings: serial.ToTypedMessage(receiverSettings),
		ProxySettings:    serial.ToTypedMessage(ts),
	}, nil
}

type OutboundDetourConfig struct {
	Protocol      string           `json:"protocol"`
	SendThrough   *Address         `json:"sendThrough"`
	Tag           string           `json:"tag"`
	Settings      *json.RawMessage `json:"settings"`
	StreamSetting *StreamConfig    `json:"streamSettings"`
	ProxySettings *ProxyConfig     `json:"proxySettings"`
	MuxSettings   *MuxConfig       `json:"mux"`
}

// Build implements Buildable.
func (c *OutboundDetourConfig) Build() (*core.OutboundHandlerConfig, error) {
	senderSettings := &proxyman.SenderConfig{}

	if c.SendThrough != nil {
		address := c.SendThrough
		if address.Family().IsDomain() {
			return nil, errors.New("unable to send through")
		}
		senderSettings.Via = address.Build()
	}

	if c.StreamSetting != nil {
		ss, err := c.StreamSetting.Build()
		if err != nil {
			return nil, err
		}
		senderSettings.StreamSettings = ss
	}

	if c.ProxySettings != nil {
		ps, err := c.ProxySettings.Build()
		if err != nil {
			return nil, errors.New("invalid outbound detour proxy settings")
		}
		senderSettings.ProxySettings = ps
	}

	if c.MuxSettings != nil {
		senderSettings.MultiplexSettings = c.MuxSettings.Build()
	}

	settings := []byte("{}")
	if c.Settings != nil {
		//fmt.Println("in infra-conf-v2raygo func (c *OutboundDetourConfig) Build() c.Setttings: ", *c.Settings)
		settings = ([]byte)(*c.Settings)
	}
	rawConfig, err := outboundConfigLoader.LoadWithID(settings, c.Protocol)
	if err != nil {
		return nil, errors.New("failed to parse to outbound detour config")
	}
	//fmt.Println("in infra-conf-v2raygo func (c *OutboundDetourConfig) Build() rawConfig.type: ", reflect.TypeOf(rawConfig))	
	ts, err := rawConfig.(Buildable).Build()
	if err != nil {
		return nil, err
	}

	return &core.OutboundHandlerConfig{
		SenderSettings: serial.ToTypedMessage(senderSettings),
		Tag:            c.Tag,
		ProxySettings:  serial.ToTypedMessage(ts),
	}, nil
}

type StatsConfig struct{}

// Build implements Buildable.
func (c *StatsConfig) Build() (*stats.Config, error) {
	return &stats.Config{}, nil
}

type Config struct {
	// Port of this Point server.
	// Deprecated: Port exists for historical compatibility
	// and should not be used.
	Port uint16 `json:"port"`

	// Deprecated: InboundConfig exists for historical compatibility
	// and should not be used.
	InboundConfig *InboundDetourConfig `json:"inbound"`

	// Deprecated: OutboundConfig exists for historical compatibility
	// and should not be used.
	OutboundConfig *OutboundDetourConfig `json:"outbound"`

	// Deprecated: InboundDetours exists for historical compatibility
	// and should not be used.
	InboundDetours []InboundDetourConfig `json:"inboundDetour"`

	// Deprecated: OutboundDetours exists for historical compatibility
	// and should not be used.
	OutboundDetours []OutboundDetourConfig `json:"outboundDetour"`

	LogConfig        *LogConfig              `json:"log"`
	RouterConfig     *RouterConfig           `json:"routing"`
	DNSConfig        *DNSConfig              `json:"dns"`
	InboundConfigs   []InboundDetourConfig   `json:"inbounds"`
	OutboundConfigs  []OutboundDetourConfig  `json:"outbounds"`
	Transport        *TransportConfig        `json:"transport"`
	Policy           *PolicyConfig           `json:"policy"`
	API              *APIConfig              `json:"api"`
	Stats            *StatsConfig            `json:"stats"`
	Reverse          *ReverseConfig          `json:"reverse"`
	FakeDNS          *FakeDNSConfig          `json:"fakeDns"`
	BrowserForwarder *BrowserForwarderConfig `json:"browserForwarder"`
}

func (c *Config) findInboundTag(tag string) int {
	found := -1
	for idx, ib := range c.InboundConfigs {
		if ib.Tag == tag {
			found = idx
			break
		}
	}
	return found
}

func (c *Config) findOutboundTag(tag string) int {
	found := -1
	for idx, ob := range c.OutboundConfigs {
		if ob.Tag == tag {
			found = idx
			break
		}
	}
	return found
}

// Override method accepts another Config overrides the current attribute
// Override 方法接受另一个 Config 覆盖当前属性
func (c *Config) Override(o *Config, fn string) {
	// only process the non-deprecated members

	if o.LogConfig != nil {
		c.LogConfig = o.LogConfig
	}
	if o.RouterConfig != nil {
		c.RouterConfig = o.RouterConfig
	}
	if o.DNSConfig != nil {
		c.DNSConfig = o.DNSConfig
	}
	if o.Transport != nil {
		c.Transport = o.Transport
	}
	if o.Policy != nil {
		c.Policy = o.Policy
	}
	if o.API != nil {
		c.API = o.API
	}
	if o.Stats != nil {
		c.Stats = o.Stats
	}
	if o.Reverse != nil {
		c.Reverse = o.Reverse
	}

	if o.FakeDNS != nil {
		c.FakeDNS = o.FakeDNS
	}

	// deprecated attrs... keep them for now
	if o.InboundConfig != nil {
		c.InboundConfig = o.InboundConfig
	}
	if o.OutboundConfig != nil {
		c.OutboundConfig = o.OutboundConfig
	}
	if o.InboundDetours != nil {
		c.InboundDetours = o.InboundDetours
	}
	if o.OutboundDetours != nil {
		c.OutboundDetours = o.OutboundDetours
	}
	// deprecated attrs

	// update the Inbound in slice if the only one in overide config has same tag
	if len(o.InboundConfigs) > 0 {
		if len(c.InboundConfigs) > 0 && len(o.InboundConfigs) == 1 {
			if idx := c.findInboundTag(o.InboundConfigs[0].Tag); idx > -1 {
				c.InboundConfigs[idx] = o.InboundConfigs[0]
				ctllog.Println("[", fn, "] updated inbound with tag: ", o.InboundConfigs[0].Tag)
			} else {
				c.InboundConfigs = append(c.InboundConfigs, o.InboundConfigs[0])
				ctllog.Println("[", fn, "] appended inbound with tag: ", o.InboundConfigs[0].Tag)
			}
		} else {
			c.InboundConfigs = o.InboundConfigs
		}
	}

	// update the Outbound in slice if the only one in overide config has same tag
	if len(o.OutboundConfigs) > 0 {
		if len(c.OutboundConfigs) > 0 && len(o.OutboundConfigs) == 1 {
			if idx := c.findOutboundTag(o.OutboundConfigs[0].Tag); idx > -1 {
				c.OutboundConfigs[idx] = o.OutboundConfigs[0]
				ctllog.Println("[", fn, "] updated outbound with tag: ", o.OutboundConfigs[0].Tag)
			} else {
				if strings.Contains(strings.ToLower(fn), "tail") {
					c.OutboundConfigs = append(c.OutboundConfigs, o.OutboundConfigs[0])
					ctllog.Println("[", fn, "] appended outbound with tag: ", o.OutboundConfigs[0].Tag)
				} else {
					c.OutboundConfigs = append(o.OutboundConfigs, c.OutboundConfigs...)
					ctllog.Println("[", fn, "] prepended outbound with tag: ", o.OutboundConfigs[0].Tag)
				}
			}
		} else {
			c.OutboundConfigs = o.OutboundConfigs
		}
	}
}

func applyTransportConfig(s *StreamConfig, t *TransportConfig) {
	if s.TCPSettings == nil {
		s.TCPSettings = t.TCPConfig
	}
	if s.KCPSettings == nil {
		s.KCPSettings = t.KCPConfig
	}
	if s.WSSettings == nil {
		s.WSSettings = t.WSConfig
	}
	if s.HTTPSettings == nil {
		s.HTTPSettings = t.HTTPConfig
	}
	if s.DSSettings == nil {
		s.DSSettings = t.DSConfig
	}
}

// Build implements Buildable.
// 构建实现可构建。
func (c *Config) Build() (*core.Config, error) {
	fmt.Println("in infa-conf-v2ray.go func (c *Config) Build()")
	// 检查 sniffing 及 dns 设置
	if err := PostProcessConfigureFile(c); err != nil {
		fmt.Println("in infa-conf-v2ray.go func (c *Config) Build() error!")
		return nil, err
	}

	// 定义一个 config 预设 调度 入站 出站 等配置选项
	config := &core.Config{
		App: []*serial.TypedMessage{
			serial.ToTypedMessage(&dispatcher.Config{}),
			serial.ToTypedMessage(&proxyman.InboundConfig{}),
			serial.ToTypedMessage(&proxyman.OutboundConfig{}),
		},
	}

	// 当 API 设置不为空时
	if c.API != nil {
		apiConf, err := c.API.Build()
		if err != nil {
			return nil, err
		}
		config.App = append(config.App, serial.ToTypedMessage(apiConf))
	}

	// 统计信息
	if c.Stats != nil {
		statsConf, err := c.Stats.Build()
		if err != nil {
			return nil, err
		}
		config.App = append(config.App, serial.ToTypedMessage(statsConf))
	}

	// 没有 log 设置时，启用默认日志设置
	var logConfMsg *serial.TypedMessage
	if c.LogConfig != nil {
		logConfMsg = serial.ToTypedMessage(c.LogConfig.Build())
	} else {
		logConfMsg = serial.ToTypedMessage(DefaultLogConfig())
	}
	// let logger module be the first App to start,
	// so that other modules could print log during initiating
	config.App = append([]*serial.TypedMessage{logConfMsg}, config.App...)

	fmt.Println("in infa-conf-v2ray.go func (c *Config) Build() 路由设置")
	// 路由设置
	if c.RouterConfig != nil {
		//gztest.GetMessageReflectType(*c.RouterConfig)
		routerConfig, err := c.RouterConfig.Build()
		if err != nil {
			return nil, err
		}
		config.App = append(config.App, serial.ToTypedMessage(routerConfig))
	}

	// dns 设置
	if c.DNSConfig != nil {
		dnsApp, err := c.DNSConfig.Build()
		if err != nil {
			return nil, errors.New("failed to parse DNS config")
		}
		config.App = append(config.App, serial.ToTypedMessage(dnsApp))
	}

	// 本地策略设置
	if c.Policy != nil {
		pc, err := c.Policy.Build()
		if err != nil {
			return nil, err
		}
		config.App = append(config.App, serial.ToTypedMessage(pc))
	}

	// 反向代理
	if c.Reverse != nil {
		r, err := c.Reverse.Build()
		if err != nil {
			return nil, err
		}
		config.App = append(config.App, serial.ToTypedMessage(r))
	}

	fmt.Println("in infa-conf-v2ray.go func (c *Config) Build() FakeDNS设置")
	if c.FakeDNS != nil {
		r, err := c.FakeDNS.Build()
		if err != nil {
			return nil, err
		}
		config.App = append(config.App, serial.ToTypedMessage(r))
	}

	// 浏览器转发模式
	if c.BrowserForwarder != nil {
		r, err := c.BrowserForwarder.Build()
		if err != nil {
			return nil, err
		}
		config.App = append(config.App, serial.ToTypedMessage(r))
	}

	var inbounds []InboundDetourConfig

	// 汇总入站设置，c.InboundConfig = inbound 单个入站设置，已弃用
	if c.InboundConfig != nil {
		inbounds = append(inbounds, *c.InboundConfig)
	}

	if len(c.InboundDetours) > 0 {
		inbounds = append(inbounds, c.InboundDetours...)
	}

	if len(c.InboundConfigs) > 0 {
		inbounds = append(inbounds, c.InboundConfigs...)
	}

	// Backward compatibility.
	// 向后兼容
	if len(inbounds) > 0 && inbounds[0].PortRange == nil && c.Port > 0 {
		inbounds[0].PortRange = &PortRange{
			From: uint32(c.Port),
			To:   uint32(c.Port),
		}
	}

	// 底层传输配置
	for _, rawInboundConfig := range inbounds {
		// 如果有全局设置，正常是没有，已弃用全局设置
		if c.Transport != nil {
			if rawInboundConfig.StreamSetting == nil {
				// 如果没有设置分协议 stream ，默认此设置各模式为空
				rawInboundConfig.StreamSetting = &StreamConfig{}
			}
			applyTransportConfig(rawInboundConfig.StreamSetting, c.Transport)
		}
		// 确定监听的地址及端口设置正确
		ic, err := rawInboundConfig.Build()
		if err != nil {
			return nil, err
		}
		config.Inbound = append(config.Inbound, ic)
	}

	var outbounds []OutboundDetourConfig

	if c.OutboundConfig != nil {
		outbounds = append(outbounds, *c.OutboundConfig)
	}

	if len(c.OutboundDetours) > 0 {
		outbounds = append(outbounds, c.OutboundDetours...)
	}

	if len(c.OutboundConfigs) > 0 {
		outbounds = append(outbounds, c.OutboundConfigs...)
	}

	for _, rawOutboundConfig := range outbounds {
		if c.Transport != nil {
			if rawOutboundConfig.StreamSetting == nil {
				rawOutboundConfig.StreamSetting = &StreamConfig{}
			}
			applyTransportConfig(rawOutboundConfig.StreamSetting, c.Transport)
		}
		oc, err := rawOutboundConfig.Build()
		if err != nil {
			return nil, err
		}
		config.Outbound = append(config.Outbound, oc)
	}
	// config 设置了 APP 及 inbounds 和 outbounds 字段
	fmt.Println("in infa-conf-v2ray.go func (c *Config) Build() return")
	return config, nil
}
