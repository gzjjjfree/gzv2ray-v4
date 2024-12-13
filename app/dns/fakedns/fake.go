// +build !confonly

package fakedns

import (
	"context"
	"math"
	"math/big"
	gonet "net"
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common"
	"github.com/gzjjjfree/gzv2ray-v4/common/cache"
	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/features/dns"
)

type Holder struct {
	domainToIP cache.Lru
	nextIP     *big.Int

	ipRange *gonet.IPNet

	config *FakeDnsPool
}

func (*Holder) Type() interface{} {
	return (*dns.FakeDNSEngine)(nil)
}

func (fkdns *Holder) Start() error {
	fmt.Println("in app--dns-fakedns-fake.go func Start()")
	return fkdns.initializeFromConfig()
}

func (fkdns *Holder) Close() error {
	fmt.Println("in app--dns-fakedns-fake.go func (fkdns *Holder) Close()")
	fkdns.domainToIP = nil
	fkdns.nextIP = nil
	fkdns.ipRange = nil
	return nil
}

func NewFakeDNSHolder() (*Holder, error) {
	var fkdns *Holder
	var err error

	if fkdns, err = NewFakeDNSHolderConfigOnly(nil); err != nil {
		return nil, errors.New("unable to create Fake Dns Engine")
	}
	err = fkdns.initialize("198.18.0.0/15", 65535)
	if err != nil {
		return nil, err
	}
	return fkdns, nil
}

func NewFakeDNSHolderConfigOnly(conf *FakeDnsPool) (*Holder, error) {
	return &Holder{nil, nil, nil, conf}, nil
}

func (fkdns *Holder) initializeFromConfig() error {
	return fkdns.initialize(fkdns.config.IpPool, int(fkdns.config.LruSize))
}

func (fkdns *Holder) initialize(ipPoolCidr string, lruSize int) error {
	fmt.Println("in app--dns-fakedns-fake.go func (fkdns *Holder) initialize")
	var ipRange *gonet.IPNet
	var ipaddr gonet.IP
	var currentIP *big.Int
	var err error

	if ipaddr, ipRange, err = gonet.ParseCIDR(ipPoolCidr); err != nil {
		return errors.New("urable to parse CIDR for Fake DNS IP assignment")
	}

	currentIP = big.NewInt(0).SetBytes(ipaddr)
	if ipaddr.To4() != nil {
		currentIP = big.NewInt(0).SetBytes(ipaddr.To4())
	}

	ones, bits := ipRange.Mask.Size()
	rooms := bits - ones
	if math.Log2(float64(lruSize)) >= float64(rooms) {
		return errors.New("uRU size is bigger than subnet size")
	}
	fkdns.domainToIP = cache.NewLru(lruSize)
	fkdns.ipRange = ipRange
	fkdns.nextIP = currentIP
	return nil
}

// GetFakeIPForDomain check and generate a fake IP for a domain name
func (fkdns *Holder) GetFakeIPForDomain(domain string) []net.Address {
	fmt.Println("in app--dns-fakedns-fake.go func (fkdns *Holder) GetFakeIPForDomain")
	if v, ok := fkdns.domainToIP.Get(domain); ok {
		return []net.Address{v.(net.Address)}
	}
	var ip net.Address
	for {
		ip = net.IPAddress(fkdns.nextIP.Bytes())

		fkdns.nextIP = fkdns.nextIP.Add(fkdns.nextIP, big.NewInt(1))
		if !fkdns.ipRange.Contains(fkdns.nextIP.Bytes()) {
			fkdns.nextIP = big.NewInt(0).SetBytes(fkdns.ipRange.IP)
		}

		// if we run for a long time, we may go back to beginning and start seeing the IP in use
		if _, ok := fkdns.domainToIP.GetKeyFromValue(ip); !ok {
			break
		}
	}
	fkdns.domainToIP.Put(domain, ip)
	return []net.Address{ip}
}

// GetDomainFromFakeDNS check if an IP is a fake IP and have corresponding domain name
func (fkdns *Holder) GetDomainFromFakeDNS(ip net.Address) string {
	fmt.Println("in app--dns-fakedns-fake.go func (fkdns *Holder) GetDomainFromFakeDNS")
	if !ip.Family().IsIP() || !fkdns.ipRange.Contains(ip.IP()) {
		return ""
	}
	if k, ok := fkdns.domainToIP.GetKeyFromValue(ip); ok {
		return k.(string)
	}
	return ""
}

func init() {
	fmt.Println("in is run ./app/dns/fakedns/fake.go func init ")
	common.Must(common.RegisterConfig((*FakeDnsPool)(nil), func(ctx context.Context, config interface{}) (interface{}, error) {
		var f *Holder
		var err error
		if f, err = NewFakeDNSHolderConfigOnly(config.(*FakeDnsPool)); err != nil {
			return nil, err
		}
		return f, nil
	}))
}
