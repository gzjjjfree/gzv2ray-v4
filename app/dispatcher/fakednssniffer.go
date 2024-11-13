// +build !confonly

package dispatcher

import (
	"context"
	"errors"
	"fmt"

	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/session"
	"github.com/gzjjjfree/gzv2ray-v4/features/dns"
)

// newFakeDNSSniffer Create a Fake DNS metadata sniffer
func newFakeDNSSniffer(ctx context.Context) (protocolSnifferWithMetadata, error) {
	var fakeDNSEngine dns.FakeDNSEngine
	err := core.RequireFeatures(ctx, func(fdns dns.FakeDNSEngine) {
		fakeDNSEngine = fdns
	})
	if err != nil {
		return protocolSnifferWithMetadata{}, err
	}
	if fakeDNSEngine == nil {
		errNotInit := errors.New("fakeDNSEngine is not initialized, but such a sniffer is used")
		return protocolSnifferWithMetadata{}, errNotInit
	}
	return protocolSnifferWithMetadata{protocolSniffer: func(ctx context.Context, bytes []byte) (SniffResult, error) {
		Target := session.OutboundFromContext(ctx).Target
		if Target.Network == net.Network_TCP || Target.Network == net.Network_UDP {
			domainFromFakeDNS := fakeDNSEngine.GetDomainFromFakeDNS(Target.Address)
			if domainFromFakeDNS != "" {
				fmt.Println("fake dns got domain: ", domainFromFakeDNS, " for ip: ", Target.Address.String())
				return &fakeDNSSniffResult{domainName: domainFromFakeDNS}, nil
			}
		}
		return nil, common.ErrNoClue
	}, metadataSniffer: true}, nil
}

type fakeDNSSniffResult struct {
	domainName string
}

func (fakeDNSSniffResult) Protocol() string {
	return "fakedns"
}

func (f fakeDNSSniffResult) Domain() string {
	return f.domainName
}