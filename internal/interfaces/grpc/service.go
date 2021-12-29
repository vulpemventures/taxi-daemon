package grpc

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/taxi-daemon/internal/application"
	"github.com/vulpemventures/taxi-daemon/internal/interfaces"
)

const (
	minPort = 1024
	maxPort = 49151
)

type service struct {
	opts  ServiceOpts
	admin handlers
	taxi  handlers
}

type ServiceConfig struct {
	Port        int
	TLSKeyPath  string
	TLSCertPath string
	Email       string
	Domain      string
}

func (c ServiceConfig) validate(checkExtraFields bool) error {
	if c.Port < minPort || c.Port > maxPort {
		return fmt.Errorf("port must be in range [%d, %d]", minPort, maxPort)
	}
	tlsKeySet, tlsCertSet := c.TLSKeyPath != "", c.TLSCertPath != ""
	if tlsKeySet != tlsCertSet {
		return fmt.Errorf(
			"both TLS key and certificate paths must be either paths to " +
				"existing files or not be defined",
		)
	}

	if checkExtraFields {
		emailSet, domainSet := c.Email != "", c.Domain != ""
		if emailSet != domainSet {
			return fmt.Errorf(
				"both email and domain must be either defined to use certmagic " +
					"or not to source TLS key/cert from existing files",
			)
		}
	}
	return nil
}

func (c ServiceConfig) insecure() bool {
	return c.TLSKeyPath == "" && c.Domain == ""
}

func (c ServiceConfig) serverAddr() string {
	return fmt.Sprintf(":%d", c.Port)
}

func (c ServiceConfig) clientAddr() string {
	return fmt.Sprintf("localhost:%d", c.Port)
}

type ServiceOpts struct {
	AdminConfig ServiceConfig
	TaxiConfig  ServiceConfig
	AdminSvc    application.AdminService
	TaxiSvc     application.TaxiService
}

func (o ServiceOpts) validate() error {
	if err := o.AdminConfig.validate(false); err != nil {
		return fmt.Errorf("invalid admin config: %s", err)
	}
	if err := o.TaxiConfig.validate(true); err != nil {
		return fmt.Errorf("invalid taxi config: %s", err)
	}
	if o.AdminSvc == nil {
		return fmt.Errorf("missing admin service")
	}
	if o.TaxiSvc == nil {
		return fmt.Errorf("missing taxi service")
	}
	return nil
}

func NewService(opts ServiceOpts) (interfaces.Service, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	adminHandlers, err := newAdminHandlers(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin handlers: %s", err)
	}
	taxiHandlers, err := newTaxiHandlers(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create taxi handlers: %s", err)
	}

	return &service{opts, *adminHandlers, *taxiHandlers}, nil
}

func (s *service) Start() error {
	log.Infof("admin server is listening on %s", s.opts.AdminConfig.serverAddr())
	if s.opts.AdminConfig.insecure() {
		go s.admin.httpServer.ListenAndServe()
	} else {
		go s.admin.httpServer.ListenAndServeTLS("", "")
	}

	log.Infof("taxi server is listening on %s", s.opts.TaxiConfig.serverAddr())
	if s.opts.AdminConfig.insecure() {
		go s.taxi.httpServer.ListenAndServe()
	} else {
		go s.taxi.httpServer.ListenAndServeTLS("", "")
	}

	return nil
}

func (s *service) Stop() {
	log.Info("stopping admin server")
	s.admin.grpc.GracefulStop()
	s.admin.httpServer.Shutdown(context.Background())

	log.Info("stopping taxi server")
	s.taxi.grpc.GracefulStop()
	s.taxi.httpServer.Shutdown(context.Background())
}
