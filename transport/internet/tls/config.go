// +build !confonly

package tls

import (
	"crypto/tls"
	"crypto/x509"
	"strings"
	"sync"
	"time"
	"errors"
	"fmt"

	"github.com/gzjjjfree/gzv2ray-v4/common/net"
	"github.com/gzjjjfree/gzv2ray-v4/common/protocol/tls/cert"
	"github.com/gzjjjfree/gzv2ray-v4/transport/internet"
)

var (
	globalSessionCache = tls.NewLRUClientSessionCache(128)
)

const exp8357 = "experiment:8357"

// ParseCertificate converts a cert.Certificate to Certificate.
func ParseCertificate(c *cert.Certificate) *Certificate {
	fmt.Println("in transport-internet-tls-config.go func ParseCertificate")
	if c != nil {
		certPEM, keyPEM := c.ToPEM()
		return &Certificate{
			Certificate: certPEM,
			Key:         keyPEM,
		}
	}
	return nil
}

func (c *Config) loadSelfCertPool() (*x509.CertPool, error) {
	fmt.Println("in transport-internet-tls-config.go func (c *Config) loadSelfCertPool()")
	root := x509.NewCertPool()
	for _, cert := range c.Certificate {
		if !root.AppendCertsFromPEM(cert.Certificate) {
			return nil, errors.New("failed to append cert")
		}
	}
	return root, nil
}

// BuildCertificates builds a list of TLS certificates from proto definition.
func (c *Config) BuildCertificates() []tls.Certificate {
	fmt.Println("in transport-internet-tls-config.go func (c *Config) BuildCertificates()")
	certs := make([]tls.Certificate, 0, len(c.Certificate))
	for _, entry := range c.Certificate {
		if entry.Usage != Certificate_ENCIPHERMENT {
			continue
		}
		keyPair, err := tls.X509KeyPair(entry.Certificate, entry.Key)
		if err != nil {
			fmt.Println("ignoring invalid X509 key pair")
			continue
		}
		certs = append(certs, keyPair)
	}
	return certs
}

func isCertificateExpired(c *tls.Certificate) bool {
	fmt.Println("in transport-internet-tls-config.go func isCertificateExpired")
	if c.Leaf == nil && len(c.Certificate) > 0 {
		if pc, err := x509.ParseCertificate(c.Certificate[0]); err == nil {
			c.Leaf = pc
		}
	}

	// If leaf is not there, the certificate is probably not used yet. We trust user to provide a valid certificate.
	return c.Leaf != nil && c.Leaf.NotAfter.Before(time.Now().Add(-time.Minute))
}

func issueCertificate(rawCA *Certificate, domain string) (*tls.Certificate, error) {
	fmt.Println("in transport-internet-tls-config.go func issueCertificate")
	parent, err := cert.ParseCertificate(rawCA.Certificate, rawCA.Key)
	if err != nil {
		return nil, errors.New("failed to parse raw certificate")
	}
	newCert, err := cert.Generate(parent, cert.CommonName(domain), cert.DNSNames(domain))
	if err != nil {
		return nil, errors.New("failed to generate new certificate for ")
	}
	newCertPEM, newKeyPEM := newCert.ToPEM()
	cert, err := tls.X509KeyPair(newCertPEM, newKeyPEM)
	return &cert, err
}

func (c *Config) getCustomCA() []*Certificate {
	certs := make([]*Certificate, 0, len(c.Certificate))
	for _, certificate := range c.Certificate {
		if certificate.Usage == Certificate_AUTHORITY_ISSUE {
			certs = append(certs, certificate)
		}
	}
	return certs
}

func getGetCertificateFunc(c *tls.Config, ca []*Certificate) func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	fmt.Println("in transport-internet-tls-config.go func getGetCertificateFunc")
	var access sync.RWMutex

	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		domain := hello.ServerName
		certExpired := false

		access.RLock()
		certificate, found := c.NameToCertificate[domain]
		access.RUnlock()

		if found {
			if !isCertificateExpired(certificate) {
				return certificate, nil
			}
			certExpired = true
		}

		if certExpired {
			newCerts := make([]tls.Certificate, 0, len(c.Certificates))

			access.Lock()
			for _, certificate := range c.Certificates {
				cert := certificate
				if !isCertificateExpired(&cert) {
					newCerts = append(newCerts, cert)
				}
			}

			c.Certificates = newCerts
			access.Unlock()
		}

		var issuedCertificate *tls.Certificate

		// Create a new certificate from existing CA if possible
		for _, rawCert := range ca {
			if rawCert.Usage == Certificate_AUTHORITY_ISSUE {
				newCert, err := issueCertificate(rawCert, domain)
				if err != nil {
					fmt.Println("failed to issue new certificate for ", domain)
					continue
				}

				access.Lock()
				c.Certificates = append(c.Certificates, *newCert)
				issuedCertificate = &c.Certificates[len(c.Certificates)-1]
				access.Unlock()
				break
			}
		}

		if issuedCertificate == nil {
			return nil, errors.New("failed to create a new certificate for ")
		}

		access.Lock()
		c.BuildNameToCertificate()
		access.Unlock()

		return issuedCertificate, nil
	}
}

func (c *Config) IsExperiment8357() bool {
	return strings.HasPrefix(c.ServerName, exp8357)
}

func (c *Config) parseServerName() string {
	if c.IsExperiment8357() {
		return c.ServerName[len(exp8357):]
	}

	return c.ServerName
}

// GetTLSConfig converts this Config into tls.Config.
func (c *Config) GetTLSConfig(opts ...Option) *tls.Config {
	fmt.Println("in transport-internet-tls-config.go func (*Config).GetTLSConfig")
	root, err := c.getCertPool()
	if err != nil {
		fmt.Println("failed to load system root certificate")
	}

	if c == nil {
		return &tls.Config{
			ClientSessionCache:     globalSessionCache,
			RootCAs:                root,
			InsecureSkipVerify:     false,
			NextProtos:             nil,
			SessionTicketsDisabled: true,
		}
	}

	config := &tls.Config{
		ClientSessionCache:     globalSessionCache,
		RootCAs:                root,
		InsecureSkipVerify:     c.AllowInsecure,
		NextProtos:             c.NextProtocol,
		SessionTicketsDisabled: !c.EnableSessionResumption,
	}

	for _, opt := range opts {
		opt(config)
	}

	config.Certificates = c.BuildCertificates()
	config.BuildNameToCertificate()

	caCerts := c.getCustomCA()
	if len(caCerts) > 0 {
		config.GetCertificate = getGetCertificateFunc(config, caCerts)
	}

	if sn := c.parseServerName(); len(sn) > 0 {
		fmt.Println("in transport-internet-tls-config.go func (*Config).GetTLSConfig config.ServerName is: ", sn)
		config.ServerName = sn
	}

	if len(config.NextProtos) == 0 {
		config.NextProtos = []string{"h2", "http/1.1"}
	}

	return config
}

// Option for building TLS config.
type Option func(*tls.Config)

// WithDestination sets the server name in TLS config.
func WithDestination(dest net.Destination) Option {
	return func(config *tls.Config) {
		if dest.Address.Family().IsDomain() && config.ServerName == "" {
			config.ServerName = dest.Address.Domain()
		}
	}
}

// WithNextProto sets the ALPN values in TLS config.
func WithNextProto(protocol ...string) Option {
	return func(config *tls.Config) {
		if len(config.NextProtos) == 0 {
			config.NextProtos = protocol
		}
	}
}

// ConfigFromStreamSettings fetches Config from stream settings. Nil if not found.
func ConfigFromStreamSettings(settings *internet.MemoryStreamConfig) *Config {
	if settings == nil {
		return nil
	}
	config, ok := settings.SecuritySettings.(*Config)
	if !ok {
		return nil
	}
	return config
}
