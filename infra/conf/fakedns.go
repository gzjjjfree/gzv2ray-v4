package conf

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/gzjjjfree/gzv2ray-v4/app/dns/fakedns"
)

type FakeDNSConfig struct {
	IPPool  string `json:"ipPool"`
	LruSize int64  `json:"poolSize"`
}

func (f FakeDNSConfig) Build() (proto.Message, error) {
	fmt.Println("in info-conf-fakedns.go func (f FakeDNSConfig) Build()")
	return &fakedns.FakeDnsPool{
		IpPool:  f.IPPool,
		LruSize: f.LruSize,
	}, nil
}

type FakeDNSPostProcessingStage struct{}

func (FakeDNSPostProcessingStage) Process(conf *Config) error {
	fmt.Println("in info-conf-fakedns.go func (FakeDNSPostProcessingStage) Process")
	var fakeDNSInUse bool

	// 当 dns 设置不为空时，
	if conf.DNSConfig != nil {
		for _, v := range conf.DNSConfig.Servers {
			// 不知怎么判断 dns 的 address 地址
			if v.Address.Family().IsDomain() {
				if v.Address.Domain() == "fakedns" {
					fakeDNSInUse = true
				}
			}
		}
	}

	// 
	if fakeDNSInUse {
		if conf.FakeDNS == nil {
			// Add a Fake DNS Config if there is none
			conf.FakeDNS = &FakeDNSConfig{
				IPPool:  "198.18.0.0/15",
				LruSize: 65535,
			}
		}
		found := false
		// Check if there is a Outbound with necessary sniffer on
		// 检查是否有一个带必要嗅探器的出站
		var inbounds []InboundDetourConfig

		// 汇总所有的入站设置
		if len(conf.InboundConfigs) > 0 {
			inbounds = append(inbounds, conf.InboundConfigs...)
		}
		for _, v := range inbounds {
			if v.SniffingConfig != nil && v.SniffingConfig.Enabled && v.SniffingConfig.DestOverride != nil {
				for _, dov := range *v.SniffingConfig.DestOverride {
					if dov == "fakedns" {
						found = true
					}
				}
			}
		}
		if !found {
			fmt.Println("Defined Fake DNS but haven't enabled fake dns sniffing at any inbound.")
		}
	}

	return nil
}
