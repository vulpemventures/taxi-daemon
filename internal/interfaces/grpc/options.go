package grpc

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"math"

	"github.com/caddyserver/certmagic"
	"github.com/vulpemventures/taxi-daemon/internal/application"
	"golang.org/x/net/http2"
)

type ServiceOpts struct {
	AdminPort   int
	TaxiPort    int
	TLSKeyPath  string
	TLSCertPath string
	Domain      string
	Email       string
	AdminSvc    application.AdminService
	TaxiSvc     application.TaxiService

	certmagicConfig *certmagic.Config
}

func (o ServiceOpts) validate() error {
	if o.AdminPort < minPort || o.AdminPort > maxPort {
		return fmt.Errorf("admin port must be in range [%d, %d]", minPort, maxPort)
	}
	if o.TaxiPort < minPort || o.TaxiPort > maxPort {
		return fmt.Errorf("taxi port must be in range [%d, %d]", minPort, maxPort)
	}
	if o.AdminPort == o.TaxiPort {
		return fmt.Errorf("admin and taxi ports must not be equal")
	}
	if math.Abs(float64(o.AdminPort-o.TaxiPort)) <= 1 {
		return fmt.Errorf("admin and taxi ports must not be 2 consecutive ports")
	}

	tlsKeySet, tlsCertSet := o.TLSKeyPath != "", o.TLSCertPath != ""
	if tlsKeySet != tlsCertSet {
		return fmt.Errorf(
			"both TLS key and certificate paths must be either paths to " +
				"existing files or not be defined",
		)
	}

	emailSet, domainSet := o.Email != "", o.Domain != ""
	if emailSet != domainSet {
		return fmt.Errorf(
			"both email and domain must be either defined to use certmagic " +
				"or not to source TLS key/cert from existing files",
		)
	}

	if o.AdminSvc == nil {
		return fmt.Errorf("missing admin service")
	}
	if o.TaxiSvc == nil {
		return fmt.Errorf("missing taxi service")
	}
	return nil
}

func (o ServiceOpts) insecure() bool {
	return o.TLSKeyPath == "" && o.Email == ""
}

func (o ServiceOpts) withCertMagic() bool {
	return o.Domain != "" && o.Email != ""
}

func (o ServiceOpts) serverAddr(port int) string {
	return fmt.Sprintf(":%d", port)
}

func (o ServiceOpts) clientAddr(port int) string {
	if o.Domain == "" {
		return fmt.Sprintf("localhost:%d", port)
	}
	return fmt.Sprintf("dns:%s:%d", o.Domain, port)
}

func (o ServiceOpts) tlsConfig() (*tls.Config, error) {
	if o.insecure() {
		return nil, nil
	}

	if o.withCertMagic() {
		if o.certmagicConfig == nil {
			cfg, err := certmagicConfig(o.Email, o.Domain)
			if err != nil {
				return nil, err
			}
			o.certmagicConfig = cfg
		}
		tlsConfig := o.certmagicConfig.TLSConfig()
		tlsConfig.NextProtos = append(
			[]string{"http/1.1", http2.NextProtoTLS, "h2-14"},
			tlsConfig.NextProtos...,
		)
		return tlsConfig, nil
	}
	cert, err := tls.LoadX509KeyPair(
		o.TLSCertPath, o.TLSKeyPath,
	)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		NextProtos:   []string{"http/1.1", http2.NextProtoTLS, "h2-14"}, // h2-14 is just for compatibility. will be eventually removed.
		Certificates: []tls.Certificate{cert},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		Rand: rand.Reader,
	}, nil

}
