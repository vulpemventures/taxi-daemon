package grpc

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/caddyserver/certmagic"
	"github.com/vulpemventures/taxi-daemon/internal/core/application"
	"golang.org/x/net/http2"
)

type ServiceOpts struct {
	Port         int
	TLSKeyPath   string
	TLSCertPath  string
	Domain       string
	Email        string
	ExternalURL  string
	AdminSwagger string
	TaxiSwagger  string
	AdminSvc     application.AdminService
	TaxiSvc      application.TaxiService

	certmagicConfig *certmagic.Config
}

func (o ServiceOpts) validate() error {
	if o.Port < minPort || o.Port > maxPort {
		return fmt.Errorf("port must be in range [%d, %d]", minPort, maxPort)
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
		return fmt.Errorf("both email and domain must be defined to use certmagic")
	}

	if o.AdminSvc == nil {
		return fmt.Errorf("missing admin service")
	}
	if o.TaxiSvc == nil {
		return fmt.Errorf("missing taxi service")
	}

	if o.AdminSwagger != "" {
		if _, err := ioutil.ReadFile(filepath.Join(o.AdminSwagger, "admin.swagger.json")); err != nil {
			return fmt.Errorf(
				"failed to open admin swagger file '%s': %s", o.AdminSwagger, err,
			)
		}
	}
	if o.TaxiSwagger != "" {
		if _, err := ioutil.ReadFile(filepath.Join(o.TaxiSwagger, "taxi.swagger.json")); err != nil {
			return fmt.Errorf(
				"failed to open taxi swagger file '%s': %s", o.TaxiSwagger, err,
			)
		}
	}
	return nil
}

func (o ServiceOpts) insecure() bool {
	return o.TLSKeyPath == "" && o.Email == ""
}

func (o ServiceOpts) withCertMagic() bool {
	return o.Domain != "" && o.Email != ""
}

func (o ServiceOpts) withSwagger() bool {
	return o.AdminSwagger != "" || o.TaxiSwagger != ""
}

func (o ServiceOpts) serverAddr() string {
	return fmt.Sprintf(":%d", o.Port)
}

func (o ServiceOpts) clientAddr() string {
	if o.ExternalURL != "" {
		return fmt.Sprintf("dns:%s", o.ExternalURL)
	}
	if o.Domain == "" {
		return fmt.Sprintf("localhost:%d", o.Port)
	}
	return fmt.Sprintf("dns:%s:%d", o.Domain, o.Port)
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
