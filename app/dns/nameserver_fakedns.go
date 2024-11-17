// +build !confonly

package dns

import (
	"context"
	"fmt"

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/features/dns"
)

type FakeDNSServer struct {
	fakeDNSEngine dns.FakeDNSEngine
}

func NewFakeDNSServer() *FakeDNSServer {
	return &FakeDNSServer{}
}

func (FakeDNSServer) Name() string {
	return "FakeDNS"
}

func (f *FakeDNSServer) QueryIP(ctx context.Context, domain string, _ net.IP, _ dns.IPOption, _ bool) ([]net.IP, error) {
	fmt.Println("in app-dns-nameserver_fakedns.go func (f *FakeDNSServer) QueryIP")
	if f.fakeDNSEngine == nil {
		if err := core.RequireFeatures(ctx, func(fd dns.FakeDNSEngine) {
			f.fakeDNSEngine = fd
		}); err != nil {
			return nil, newError("Unable to locate a fake DNS Engine").Base(err).AtError()
		}
	}
	ips := f.fakeDNSEngine.GetFakeIPForDomain(domain)

	netIP, err := toNetIP(ips)
	if err != nil {
		return nil, newError("Unable to convert IP to net ip").Base(err).AtError()
	}

	newError(f.Name(), " got answer: ", domain, " -> ", ips).AtInfo().WriteToLog()

	return netIP, nil
}
