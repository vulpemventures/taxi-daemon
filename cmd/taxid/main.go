package main

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/application"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/dbbadger"
	"github.com/vulpemventures/liquid-taxi/internal/interfaces/grpc"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
	"golang.org/x/net/http2"
	g "google.golang.org/grpc"
)

func main() {
	log.Info("---- Liquid Taxi ----")

	log.Info("loading datadir...")

	datadirPath := config.GetString(config.DatadirKey)

	err := dbbadger.CreateDirIfNotExist(datadirPath)
	if err != nil {
		log.Fatal(err)
	}

	dbLogger := log.New()
	hdTreeRepository, err := dbbadger.NewBadgerHDTreeRepository(datadirPath, dbLogger)
	if err != nil {
		log.Fatal(err)
	}
	topupRepository, err := dbbadger.NewBadgerTopupRepository(datadirPath, dbLogger)
	if err != nil {
		log.Fatal(err)
	}
	lockRepository, err := dbbadger.NewBadgerLockRepository(datadirPath, dbLogger)
	if err != nil {
		log.Fatal(err)
	}
	priceRepository, err := dbbadger.NewBadgerPriceRepository(datadirPath, dbLogger)
	if err != nil {
		log.Fatal(err)
	}
	explorerSvc := explorer.NewService(config.GetString(config.ExplorerEndpointKey))

	log.Info("Taxi server is starting...")

	argsTaviSvc := application.NewTaxiServiceArgs{
		HdTreeRepo:     hdTreeRepository,
		LockRepo:       lockRepository,
		TopupRepo:      topupRepository,
		PriceRepo:      priceRepository,
		ExplorerSvc:    explorerSvc,
		UtxoTTL:        time.Duration(config.GetInt(config.UtxoTTLKey)) * time.Second,
		Network:        *config.GetNetwork(),
		FixedFeeAmount: uint64(config.GetInt(config.FixedFeeAmountKey)),
		Mnemonic:       config.GetMnemonic(),
	}

	taxiSvc := application.NewTaxiService(argsTaviSvc)

	adminSvc := application.NewAdminService(
		hdTreeRepository,
		topupRepository,
		priceRepository,
		explorerSvc,
		config.GetMnemonic(),
		*config.GetNetwork(),
	)

	taxiHandler := grpc.NewTaxiHandler(taxiSvc)
	adminHandler := grpc.NewAdminHandler(adminSvc)

	taxiServer := g.NewServer()
	adminServer := g.NewServer()

	admin.RegisterAdminServer(adminServer, adminHandler)
	pb.RegisterTaxiServer(taxiServer, taxiHandler)

	domainURL := "localhost"
	taxiAddress := fmt.Sprintf("%s:%+v", domainURL, config.GetInt(config.TaxiPortKey))
	adminAddress := fmt.Sprintf("%s:%+v", domainURL, config.GetInt(config.AdminPortKey))

	if err := serveMux(taxiAddress, true, taxiServer); err != nil {
		log.WithError(err).Panic("error listening on taxi interface")
	}

	log.Info("taxi server is listening on ", taxiAddress)

	if err := serveMux(adminAddress, false, adminServer); err != nil {
		log.WithError(err).Panic("error listening on admin interface")
	}

	log.Info("admin server is listening on ", adminAddress)

	// Interrupt Notification.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-interrupt
	fmt.Println("")
	log.Info("Shutting down Taxi service...")
	adminServer.Stop()
	log.Info("Admin server stopped")
	taxiServer.Stop()
	log.Info("Taxi server stopped")

	os.Exit(0)
}

func serveMux(address string, withSsl bool, grpcServer *g.Server) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	if sslKey := config.GetString(config.SSLKeyPathKey); sslKey != "" && withSsl {
		certificate, err := tls.LoadX509KeyPair(config.GetString(config.SSLCertPathKey), sslKey)
		if err != nil {
			return err
		}

		const requiredCipher = tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
		config := &tls.Config{
			CipherSuites: []uint16{requiredCipher},
			NextProtos:   []string{"http/1.1", http2.NextProtoTLS, "h2-14"}, // h2-14 is just for compatibility. will be eventually removed.
			Certificates: []tls.Certificate{certificate},
		}
		config.Rand = rand.Reader

		lis = tls.NewListener(lis, config)
	}

	mux := cmux.New(lis)
	grpcL := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
	httpL := mux.Match(cmux.HTTP1Fast())

	grpcWebServer := grpcweb.WrapServer(
		grpcServer,
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)

	go grpcServer.Serve(grpcL)
	go http.Serve(httpL, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if isValidRequest(req) {
			grpcWebServer.ServeHTTP(resp, req)
		}
	}))

	go mux.Serve()
	return nil
}

func isValidRequest(req *http.Request) bool {
	return isValidGrpcWebOptionRequest(req) || isValidGrpcWebRequest(req)
}

func isValidGrpcWebRequest(req *http.Request) bool {
	return req.Method == http.MethodPost && isValidGrpcContentTypeHeader(req.Header.Get("content-type"))
}

func isValidGrpcContentTypeHeader(contentType string) bool {
	return strings.HasPrefix(contentType, "application/grpc-web-text") ||
		strings.HasPrefix(contentType, "application/grpc-web")
}

func isValidGrpcWebOptionRequest(req *http.Request) bool {
	accessControlHeader := req.Header.Get("Access-Control-Request-Headers")
	return req.Method == http.MethodOptions &&
		strings.Contains(accessControlHeader, "x-grpc-web") &&
		strings.Contains(accessControlHeader, "content-type")
}
