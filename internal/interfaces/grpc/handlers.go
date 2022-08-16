package grpc

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	admin "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/taxi-daemon/v1"
	taxi "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/taxi/v1"
	"github.com/vulpemventures/taxi-daemon/internal/interfaces/grpc/handler"
	"github.com/vulpemventures/taxi-daemon/internal/interfaces/grpc/interceptor"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type handlers struct {
	grpc        *grpc.Server
	grpcweb     *grpcweb.WrappedGrpcServer
	grpcgateway http.Handler
	httpServer  *http.Server
}

func newAdminHandlers(opts ServiceOpts, tlsConfig *tls.Config) (*handlers, error) {
	// grpc server for admin service
	serverOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.UnaryLogger),
		grpc.StreamInterceptor(interceptor.StreamLogger),
	}
	if tlsConfig != nil {
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	grpcHandler := handler.NewAdminHandler(opts.AdminSvc)
	grpcServer := grpc.NewServer(serverOpts...)
	admin.RegisterAdminServiceServer(grpcServer, grpcHandler)

	// grpcweb wrapped server
	grpcWebServer := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)

	// grpc gateway
	dialOpts := make([]grpc.DialOption, 0)
	if opts.insecure() {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} else {
		tlsCreds := credentials.NewTLS(nil)
		if !opts.withCertMagic() {
			tlsCreds, _ = credentials.NewClientTLSFromFile(opts.TLSCertPath, "")
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(tlsCreds))
	}
	conn, err := grpc.DialContext(
		context.Background(), opts.adminClientAddr(), dialOpts...,
	)
	if err != nil {
		return nil, err
	}
	gwmux := runtime.NewServeMux()
	admin.RegisterAdminServiceHandler(context.Background(), gwmux, conn)
	grpcGateway := http.Handler(gwmux)

	// http server routing requests to either grpc|grpcweb|grpcgateway handler
	// based on their header
	handler := router(grpcServer, grpcWebServer, grpcGateway)
	if opts.insecure() {
		handler = h2c.NewHandler(handler, &http2.Server{})
	}

	httpServer := &http.Server{
		Addr:      opts.serverAddr(opts.AdminPort),
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	return &handlers{grpcServer, grpcWebServer, grpcGateway, httpServer}, nil
}

func newTaxiHandlers(opts ServiceOpts, tlsConfig *tls.Config) (*handlers, error) {
	// grpc server for taxi service
	serverOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.UnaryLogger),
		grpc.StreamInterceptor(interceptor.StreamLogger),
	}

	if tlsConfig != nil {
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	grpcHandler := handler.NewTaxiHandler(opts.TaxiSvc)
	grpcServer := grpc.NewServer(serverOpts...)
	taxi.RegisterTaxiServiceServer(grpcServer, grpcHandler)

	// grpcweb wrapped server
	grpcWebServer := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)

	// grpc gateway
	dialOpts := make([]grpc.DialOption, 0)
	if opts.insecure() {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} else {
		tlsCreds := credentials.NewTLS(nil)
		if !opts.withCertMagic() {
			tlsCreds, _ = credentials.NewClientTLSFromFile(opts.TLSCertPath, "")
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(tlsCreds))
	}
	conn, err := grpc.DialContext(
		context.Background(), opts.taxiClientAddr(), dialOpts...,
	)
	if err != nil {
		return nil, err
	}
	gwmux := runtime.NewServeMux()
	taxi.RegisterTaxiServiceHandler(context.Background(), gwmux, conn)
	grpcGateway := http.Handler(gwmux)

	// http server routing requests to either grpc|grpcweb|grpcgateway handler
	// based on their header
	handler := router(grpcServer, grpcWebServer, grpcGateway)
	if opts.insecure() {
		handler = h2c.NewHandler(handler, &http2.Server{})
	}

	httpServer := &http.Server{
		Addr:      opts.serverAddr(opts.TaxiPort),
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	return &handlers{grpcServer, grpcWebServer, grpcGateway, httpServer}, nil
}

func (h handlers) start(withInsecure bool) {
	if withInsecure {
		go h.httpServer.ListenAndServe()
		return
	}

	go h.httpServer.ListenAndServeTLS("", "")
}

func (h handlers) stop() {
	h.httpServer.Shutdown(context.Background())
}
