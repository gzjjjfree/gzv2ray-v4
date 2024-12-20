package inbound

import (
	"context"
	"errors"
	"fmt"
	//"reflect"
	//"example.com/gztest"

	//"example.com/gztest"
	core "github.com/gzjjjfree/gzv2ray-v4"
	"github.com/gzjjjfree/gzv2ray-v4/app/proxyman"
	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/dice"
	"github.com/gzjjjfree/gzv2ray-v4/common/mux"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/features/policy"
	"github.com/gzjjjfree/gzv2ray-v4/features/stats"
	"github.com/gzjjjfree/gzv2ray-v4/proxy"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
)


func getStatCounter(v *core.Instance, tag string) (stats.Counter, stats.Counter) {
	var uplinkCounter stats.Counter
	var downlinkCounter stats.Counter

	policy := v.GetFeature(policy.ManagerType()).(policy.Manager)
	if len(tag) > 0 && policy.ForSystem().Stats.InboundUplink {
		statsManager := v.GetFeature(stats.ManagerType()).(stats.Manager)
		name := "inbound>>>" + tag + ">>>traffic>>>uplink"
		c, _ := stats.GetOrRegisterCounter(statsManager, name)
		if c != nil {
			uplinkCounter = c
		}
	}
	if len(tag) > 0 && policy.ForSystem().Stats.InboundDownlink {
		statsManager := v.GetFeature(stats.ManagerType()).(stats.Manager)
		name := "inbound>>>" + tag + ">>>traffic>>>downlink"
		c, _ := stats.GetOrRegisterCounter(statsManager, name)
		if c != nil {
			downlinkCounter = c
		}
	}

	return uplinkCounter, downlinkCounter
}

type AlwaysOnInboundHandler struct {
	proxy   proxy.Inbound
	workers []worker
	mux     *mux.Server
	tag     string
}

func NewAlwaysOnInboundHandler(ctx context.Context, tag string, receiverConfig *proxyman.ReceiverConfig, proxyConfig interface{}) (*AlwaysOnInboundHandler, error) {
	fmt.Println("in app-proxyman-inbound-always.go func NewAlwaysOnInboundHandler ctx: ", ctx)
	rawProxy, err := common.CreateObject(ctx, proxyConfig)
	if err != nil {
		return nil, err
	}
	p, ok := rawProxy.(proxy.Inbound)
	if !ok {
		return nil, errors.New("not an inbound proxy.")
	}

	h := &AlwaysOnInboundHandler{
		proxy: p,
		mux:   mux.NewServer(ctx),
		tag:   tag,
	}
	//fmt.Println("the mux.NewServer type is: %T", reflect.TypeOf(h.mux))
	//fmt.Println("the mux.NewServer type is: %T", h.mux)
	//gztest.GetMessageReflectType(*h.mux)
	uplinkCounter, downlinkCounter := getStatCounter(core.MustFromContext(ctx), tag)

	nl := p.Network()
	pr := receiverConfig.PortRange
	address := receiverConfig.Listen.AsAddress()
	if address == nil {
		address = net.AnyIP
	}

	mss, err := internet.ToMemoryStreamConfig(receiverConfig.StreamSettings)
	if err != nil {
		return nil, errors.New("failed to parse stream config")
	}

	if receiverConfig.ReceiveOriginalDestination {
		if mss.SocketSettings == nil {
			mss.SocketSettings = &internet.SocketConfig{}
		}
		if mss.SocketSettings.Tproxy == internet.SocketConfig_Off {
			mss.SocketSettings.Tproxy = internet.SocketConfig_Redirect
		}
		mss.SocketSettings.ReceiveOriginalDestAddress = true
	}
	if pr == nil {
		if net.HasNetwork(nl, net.Network_UNIX) {
			errors.New("creating unix domain socket worker on ")

			worker := &dsWorker{
				address:         address,
				proxy:           p,
				stream:          mss,
				tag:             tag,
				dispatcher:      h.mux,
				sniffingConfig:  receiverConfig.GetEffectiveSniffingSettings(),
				uplinkCounter:   uplinkCounter,
				downlinkCounter: downlinkCounter,
				ctx:             ctx,
			}
			h.workers = append(h.workers, worker)
		}
	}
	if pr != nil {
		for port := pr.From; port <= pr.To; port++ {
			if net.HasNetwork(nl, net.Network_TCP) {
				errors.New("creating stream worker on ")

				worker := &tcpWorker{
					address:         address,
					port:            net.Port(port),
					proxy:           p,
					stream:          mss,
					recvOrigDest:    receiverConfig.ReceiveOriginalDestination,
					tag:             tag,
					dispatcher:      h.mux,
					sniffingConfig:  receiverConfig.GetEffectiveSniffingSettings(),
					uplinkCounter:   uplinkCounter,
					downlinkCounter: downlinkCounter,
					ctx:             ctx,
				}
				h.workers = append(h.workers, worker)
			}

			if net.HasNetwork(nl, net.Network_UDP) {
				worker := &udpWorker{
					ctx:             ctx,
					tag:             tag,
					proxy:           p,
					address:         address,
					port:            net.Port(port),
					dispatcher:      h.mux,
					sniffingConfig:  receiverConfig.GetEffectiveSniffingSettings(),
					uplinkCounter:   uplinkCounter,
					downlinkCounter: downlinkCounter,
					stream:          mss,
				}
				h.workers = append(h.workers, worker)
			}
		}
	}

	return h, nil
}

// Start implements common.Runnable.
func (h *AlwaysOnInboundHandler) Start() error {
	fmt.Println("in app-proxyman-inbound-always.go func Start()")
	for _, worker := range h.workers {
		if err := worker.Start(); err != nil {
			return err
		}
	}
	return nil
}

// Close implements common.Closable.
func (h *AlwaysOnInboundHandler) Close() error {
	fmt.Println("in app-proxyman-inbound-always.go func Close()")
	var errs []error
	for _, worker := range h.workers {
		errs = append(errs, worker.Close())
	}
	errs = append(errs, h.mux.Close())
	if err := fmt.Errorf("error2: [%w]", errs); errors.Unwrap(err) != nil {
		return errors.New("failed to close all resources")
	}
	return nil
}

func (h *AlwaysOnInboundHandler) GetRandomInboundProxy() (interface{}, net.Port, int) {
	if len(h.workers) == 0 {
		return nil, 0, 0
	}
	w := h.workers[dice.Roll(len(h.workers))]
	return w.Proxy(), w.Port(), 9999
}

func (h *AlwaysOnInboundHandler) Tag() string {
	return h.tag
}

func (h *AlwaysOnInboundHandler) GetInbound() proxy.Inbound {
	return h.proxy
}
