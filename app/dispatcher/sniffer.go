// +build !confonly

package dispatcher

import (
	"context"
	"errors"
	"fmt"
	//"example.com/gztest"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/protocol/bittorrent"
	"github.com/gzjjjfree/gzv2ray-v4/common/protocol/http"
	"github.com/gzjjjfree/gzv2ray-v4/common/protocol/tls"
)

type SniffResult interface {
	Protocol() string
	Domain() string
}

type protocolSniffer func(context.Context, []byte) (SniffResult, error)

type protocolSnifferWithMetadata struct {
	protocolSniffer protocolSniffer
	// A Metadata sniffer will be invoked on connection establishment only, with nil body,
	// for both TCP and UDP connections
	// It will not be shown as a traffic type for routing unless there is no other successful sniffing.
	metadataSniffer bool
}

type Sniffer struct {
	sniffer []protocolSnifferWithMetadata
}

func NewSniffer(ctx context.Context) *Sniffer {
	fmt.Println("in app-dispatcher-snifffer.go func NewSniffer")
	ret := &Sniffer{
		sniffer: []protocolSnifferWithMetadata{
			{func(c context.Context, b []byte) (SniffResult, error) { return http.SniffHTTP(b) }, false},
			{func(c context.Context, b []byte) (SniffResult, error) { return tls.SniffTLS(b) }, false},
			{func(c context.Context, b []byte) (SniffResult, error) { return bittorrent.SniffBittorrent(b) }, false},
		},
	}
	//if sniffer, err := newFakeDNSSniffer(ctx); err == nil {
	//	ret.sniffer = append(ret.sniffer, sniffer)
	//}
	return ret
}

var errUnknownContent = errors.New("unknown content")

func (s *Sniffer) Sniff(c context.Context, payload []byte) (SniffResult, error) {
	fmt.Println("in app-dispatcher-snifffer.go func (s *Sniffer) Sniff")
	var pendingSniffer []protocolSnifferWithMetadata
	for _, si := range s.sniffer {
		s := si.protocolSniffer
		if si.metadataSniffer {
			continue
		}
		result, err := s(c, payload)
		if err == common.ErrNoClue {
			pendingSniffer = append(pendingSniffer, si)
			continue
		}

		if err == nil && result != nil {
			return result, nil
		}
	}

	if len(pendingSniffer) > 0 {
		s.sniffer = pendingSniffer
		return nil, common.ErrNoClue
	}

	return nil, errUnknownContent
}

func (s *Sniffer) SniffMetadata(c context.Context) (SniffResult, error) {
	fmt.Println("in app-dispatcher-snifffer.go func (s *Sniffer) SniffMetadata")
	var pendingSniffer []protocolSnifferWithMetadata
	for _, si := range s.sniffer {
		//gztest.GetMessageReflectType(si)
		s := si.protocolSniffer
		if !si.metadataSniffer {
			pendingSniffer = append(pendingSniffer, si)
			continue
		}
		fmt.Println("in app-dispatcher-snifffer.go func (s *Sniffer) SniffMetadata after continue")
		result, err := s(c, nil)
		if err == common.ErrNoClue {
			pendingSniffer = append(pendingSniffer, si)
			continue
		}

		if err == nil && result != nil {
			return result, nil
		}
	}

	if len(pendingSniffer) > 0 {
		s.sniffer = pendingSniffer
		return nil, common.ErrNoClue
	}

	return nil, errUnknownContent
}

func CompositeResult(domainResult SniffResult, protocolResult SniffResult) SniffResult {
	return &compositeResult{domainResult: domainResult, protocolResult: protocolResult}
}

type compositeResult struct {
	domainResult   SniffResult
	protocolResult SniffResult
}

func (c compositeResult) Protocol() string {
	return c.protocolResult.Protocol()
}

func (c compositeResult) Domain() string {
	return c.domainResult.Domain()
}

func (c compositeResult) ProtocolForDomainResult() string {
	return c.domainResult.Protocol()
}

type SnifferResultComposite interface {
	ProtocolForDomainResult() string
}
