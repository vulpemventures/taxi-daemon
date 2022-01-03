package grpc

import (
	"fmt"

	log "github.com/sirupsen/logrus"
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

func NewService(opts ServiceOpts) (interfaces.Service, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	tlsConfig, err := opts.tlsConfig()
	if err != nil {
		return nil, err
	}

	adminHandlers, err := newAdminHandlers(opts, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin handlers: %s", err)
	}
	taxiHandlers, err := newTaxiHandlers(opts, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create taxi handlers: %s", err)
	}

	return &service{opts, *adminHandlers, *taxiHandlers}, nil
}

func (s *service) Start() error {
	log.Infof("admin server is listening on %s", s.opts.serverAddr(s.opts.AdminPort))
	s.admin.start(s.opts.insecure())

	log.Infof("taxi server is listening on %s", s.opts.serverAddr(s.opts.TaxiPort))
	s.taxi.start(s.opts.insecure())

	return nil
}

func (s *service) Stop() {
	log.Info("stopping admin server")
	s.admin.stop()

	log.Info("stopping taxi server")
	s.taxi.stop()
}
