package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	log "github.com/sirupsen/logrus"
	admin "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/taxi-daemon/v1"
	taxi "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/taxi/v1"
	"github.com/vulpemventures/taxi-daemon/internal/interfaces"
	"github.com/vulpemventures/taxi-daemon/internal/interfaces/grpc/handler"
	"github.com/vulpemventures/taxi-daemon/internal/interfaces/grpc/interceptor"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	minPort         = 1024
	maxPort         = 49151
	swaggerBasePath = "api-spec/protobuf/gen/swagger"
)

var (
	taxiSpecPath  = http.Dir(fmt.Sprintf("%s/taxi/v1", swaggerBasePath))
	adminSpecPath = http.Dir(fmt.Sprintf("%s/taxi-daemon/v1", swaggerBasePath))
)

type service struct {
	opts   ServiceOpts
	server *http.Server
}

func NewService(opts ServiceOpts) (interfaces.Service, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	tlsConfig, err := opts.tlsConfig()
	if err != nil {
		return nil, err
	}

	svr, err := newServer(opts, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin handlers: %s", err)
	}

	return &service{opts, svr}, nil
}

func (s *service) Start() error {
	log.Infof("server is listening on %s", s.opts.serverAddr())
	if s.opts.insecure() {
		go s.server.ListenAndServe()
		return nil
	}

	go s.server.ListenAndServeTLS("", "")
	return nil
}

func (s *service) Stop() {
	log.Info("stopping server")
	s.server.Shutdown(context.Background())
}

func newServer(opts ServiceOpts, tlsConfig *tls.Config) (*http.Server, error) {
	// grpc server for admin service
	serverOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.UnaryLogger),
		grpc.StreamInterceptor(interceptor.StreamLogger),
	}
	if tlsConfig != nil {
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	adminHandler := handler.NewAdminHandler(opts.AdminSvc)
	taxiHandler := handler.NewTaxiHandler(opts.TaxiSvc)
	grpcServer := grpc.NewServer(serverOpts...)
	admin.RegisterAdminServiceServer(grpcServer, adminHandler)
	taxi.RegisterTaxiServiceServer(grpcServer, taxiHandler)

	// grpcweb wrapped server
	grpcWebServer := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)

	// grpc gateway
	dialOpts := make([]grpc.DialOption, 0)
	if opts.insecure() {
		creds := insecure.NewCredentials()
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		tlsCreds := credentials.NewTLS(nil)
		if !opts.withCertMagic() {
			tlsCreds, _ = credentials.NewClientTLSFromFile(opts.TLSCertPath, "")
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(tlsCreds))
	}
	conn, err := grpc.DialContext(
		context.Background(), opts.clientAddr(), dialOpts...,
	)
	if err != nil {
		return nil, err
	}
	gwmux := runtime.NewServeMux()
	admin.RegisterAdminServiceHandler(context.Background(), gwmux, conn)
	taxi.RegisterTaxiServiceHandler(context.Background(), gwmux, conn)
	grpcGateway := http.Handler(gwmux)

	// http server routing requests to either grpc|grpcweb|grpcgateway handler
	// based on their header
	handler := router(grpcServer, grpcWebServer, grpcGateway)
	mux := http.NewServeMux()
	mux.Handle("/", handler)

	// swagger json file
	mux.Handle("/admin.swagger.json", http.FileServer(adminSpecPath))
	mux.Handle("/taxi.swagger.json", http.FileServer(taxiSpecPath))
	// swagger API doc
	adminSwOpts := middleware.SwaggerUIOpts{
		SpecURL: "/taxi.swagger.json",
		Path:    "/v1/docs",
	}
	adminSh := middleware.SwaggerUI(adminSwOpts, nil)
	mux.Handle("/v1/docs", adminSh)

	httpServerHandler := http.Handler(mux)
	if opts.insecure() {
		httpServerHandler = h2c.NewHandler(httpServerHandler, &http2.Server{})
	}

	return &http.Server{
		Addr:      opts.serverAddr(),
		Handler:   httpServerHandler,
		TLSConfig: tlsConfig,
	}, nil
}
