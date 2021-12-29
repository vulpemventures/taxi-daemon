package grpc

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/admin"
	"github.com/vulpemventures/taxi-daemon/internal/interfaces/grpc/handler"
	"github.com/vulpemventures/taxi-daemon/internal/interfaces/grpc/interceptor"
	"github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
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

func newAdminHandlers(opts ServiceOpts) (*handlers, error) {
	// grpc server for admin service
	serverOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.UnaryLogger),
		grpc.StreamInterceptor(interceptor.StreamLogger),
	}
	if !opts.AdminConfig.insecure() {
		creds, err := credentials.NewServerTLSFromFile(
			opts.AdminConfig.TLSCertPath, opts.AdminConfig.TLSKeyPath,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create TLS credentials for admin svc: %s", err,
			)
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}
	grpcHandler := handler.NewAdminHandler(opts.AdminSvc)
	grpcServer := grpc.NewServer(serverOpts...)
	admin.RegisterAdminServer(grpcServer, grpcHandler)

	// grpcweb wrapped server
	grpcWebServer := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)

	// grpc gateway
	dialOpts := make([]grpc.DialOption, 0)
	if opts.AdminConfig.insecure() {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} else {
		tlsCreds, _ := credentials.NewClientTLSFromFile(opts.AdminConfig.TLSCertPath, "")
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(tlsCreds))
	}
	conn, err := grpc.DialContext(
		context.Background(), opts.AdminConfig.clientAddr(), dialOpts...,
	)
	if err != nil {
		return nil, err
	}
	gwmux := runtime.NewServeMux()
	admin.RegisterAdminHandler(context.Background(), gwmux, conn)
	grpcGateway := http.Handler(gwmux)

	// http server routing requests to either grpc|grpcweb|grpcgateway handler
	// based on their header
	var tlsConfig *tls.Config
	if !opts.AdminConfig.insecure() {
		cert, _ := tls.LoadX509KeyPair(opts.AdminConfig.TLSCertPath, opts.AdminConfig.TLSKeyPath)
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}
	handler := router(grpcServer, grpcWebServer, grpcGateway)
	if opts.AdminConfig.insecure() {
		handler = h2c.NewHandler(handler, &http2.Server{})
	}
	httpServer := &http.Server{
		Addr:      opts.AdminConfig.serverAddr(),
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	return &handlers{grpcServer, grpcWebServer, grpcGateway, httpServer}, nil
}

func newTaxiHandlers(opts ServiceOpts) (*handlers, error) {
	// grpc server for taxi
	serverOpts := []grpc.ServerOption{
		grpc.UnaryInterceptor(interceptor.UnaryLogger),
		grpc.StreamInterceptor(interceptor.StreamLogger),
	}
	withInsecure := opts.TaxiConfig.insecure()
	withCertMagic := !withInsecure && opts.TaxiConfig.Email != ""
	var tlsConfig *tls.Config

	if !withInsecure {
		if withCertMagic {
			var err error
			tlsConfig, err = credsFromCertMagic(
				opts.TaxiConfig.Email, opts.TaxiConfig.Domain,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to create TLS credentials for taxi svc: %s", err,
				)
			}
		} else {
			cert, err := tls.LoadX509KeyPair(
				opts.TaxiConfig.TLSCertPath, opts.TaxiConfig.TLSKeyPath,
			)
			if err != nil {
				return nil, err
			}
			tlsConfig = &tls.Config{
				NextProtos:   []string{"http/1.1", http2.NextProtoTLS, "h2-14"}, // h2-14 is just for compatibility. will be eventually removed.
				Certificates: []tls.Certificate{cert},
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				},
				Rand: rand.Reader,
			}
		}
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	grpcHandler := handler.NewTaxiHandler(opts.TaxiSvc)
	grpcServer := grpc.NewServer(serverOpts...)
	taxi.RegisterTaxiServer(grpcServer, grpcHandler)

	// grpcweb wrapped server
	grpcWebServer := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)

	// grpc gateway
	dialOpts := make([]grpc.DialOption, 0)
	if opts.TaxiConfig.insecure() {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	} else {
		tlsCreds := credentials.NewTLS(nil)
		if !withCertMagic {
			tlsCreds, _ = credentials.NewClientTLSFromFile(opts.TaxiConfig.TLSCertPath, "")
		}
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(tlsCreds))
	}
	conn, err := grpc.DialContext(
		context.Background(), opts.TaxiConfig.clientAddr(), dialOpts...,
	)
	if err != nil {
		return nil, err
	}
	gwmux := runtime.NewServeMux()
	taxi.RegisterTaxiHandler(context.Background(), gwmux, conn)
	grpcGateway := http.Handler(gwmux)

	// http server routing requests to either grpc|grpcweb|grpcgateway handler
	// based on their header
	handler := router(grpcServer, grpcWebServer, grpcGateway)
	if withInsecure {
		handler = h2c.NewHandler(handler, &http2.Server{})
	}

	httpServer := &http.Server{
		Addr:      opts.TaxiConfig.serverAddr(),
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	return &handlers{grpcServer, grpcWebServer, grpcGateway, httpServer}, nil
}
