package main

import (
	"context"
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

	"github.com/caddyserver/certmagic"
	"github.com/go-acme/lego/v3/challenge/tlsalpn01"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	explorer "github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/admin"
	"github.com/vulpemventures/taxi-daemon/config"
	"github.com/vulpemventures/taxi-daemon/internal/application"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/dbbadger"
	"github.com/vulpemventures/taxi-daemon/internal/interfaces/grpc"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
	"golang.org/x/net/http2"
	g "google.golang.org/grpc"
)

func main() {
	log.Info("---- Liquid Taxi ----")

	datadirPath := config.GetString(config.DatadirKey)
	network := *config.GetNetwork()
	mnemonic := config.GetMnemonic()
	utxoTTL := time.Duration(config.GetInt(config.UtxoTTLKey)) * time.Second
	cacheInterval := time.Duration(config.GetInt(config.UtxoCacheIntervalKey)) * time.Second
	fixedFeeAmount := uint64(config.GetInt(config.FixedFeeAmountKey))

	log.Info("loading datadir... ", datadirPath)
	err := dbbadger.CreateDirIfNotExist(datadirPath)
	if err != nil {
		log.Fatal(err)
	}

	dbLogger := log.New()
	hdTreeRepository, err := dbbadger.NewBadgerHDTreeRepository(datadirPath, mnemonic, network, dbLogger)
	if err != nil {
		log.Fatal(err)
	}
	topupRepository, err := dbbadger.NewBadgerTopupRepository(datadirPath, dbLogger)
	if err != nil {
		log.Fatal(err)
	}

	lockRepository, err := dbbadger.NewBadgerLockRepository(datadirPath, utxoTTL, dbLogger)
	if err != nil {
		log.Fatal(err)
	}
	priceRepository, err := dbbadger.NewBadgerPriceRepository(datadirPath, dbLogger)
	if err != nil {
		log.Fatal(err)
	}
	explorerSvc, err := explorer.NewService(config.GetString(config.ExplorerEndpointKey))
	if err != nil {
		log.Fatal(err)
	}

	log.Info("taxi server is starting...")

	cache := application.NewCache(application.NewCacheArgs{
		Explorer:         explorerSvc,
		HdTreeRepository: hdTreeRepository,
		LockRepository:   lockRepository,
		FragmentAmount:   fixedFeeAmount,
		FragmentAsset:    network.AssetID,
		Interval:         cacheInterval,
	})

	argsTaviSvc := application.NewTaxiServiceArgs{
		HdTreeRepo:     hdTreeRepository,
		LockRepo:       lockRepository,
		TopupRepo:      topupRepository,
		PriceRepo:      priceRepository,
		Network:        network,
		FixedFeeAmount: fixedFeeAmount,
		Mnemonic:       mnemonic,
		Cache:          cache,
	}

	taxiSvc := application.NewTaxiService(argsTaviSvc)

	adminSvcArgs := application.NewAdminServiceArgs{
		HDTreeRepository: hdTreeRepository,
		TopupRepository:  topupRepository,
		PriceRepository:  priceRepository,
		LockRepository:   lockRepository,
		ExplorerService:  explorerSvc,
		Mnemonic:         mnemonic,
		Network:          network,
		Cache:            cache,
	}

	adminSvc := application.NewAdminService(adminSvcArgs)

	taxiHandler := grpc.NewTaxiHandler(taxiSvc)
	adminHandler := grpc.NewAdminHandler(adminSvc)

	taxiServer := g.NewServer()
	adminServer := g.NewServer()

	admin.RegisterAdminServer(adminServer, adminHandler)
	pb.RegisterTaxiServer(taxiServer, taxiHandler)

	taxiAddress := fmt.Sprintf(":%+v", config.GetInt(config.TaxiPortKey))
	adminAddress := fmt.Sprintf(":%+v", config.GetInt(config.AdminPortKey))

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
	log.Info("shutting down Taxi service...")
	adminServer.Stop()
	log.Info("admin server stopped")
	taxiServer.Stop()
	log.Info("taxi server stopped")

	os.Exit(0)
}

func serveMux(address string, withSsl bool, grpcServer *g.Server) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	// If the DOMAIN and EMAIL is given, we are going to spinup a basic http server to handle the http ACME challenge
	if domain := config.GetString(config.DomainKey); domain != "" && withSsl {

		magic := certmagic.NewDefault()
		myACME := certmagic.NewACMEManager(magic, certmagic.ACMEManager{
			Agreed: true,
			Email:  config.GetString(config.EmailKey),
		})
		magic.Issuer = myACME

		// we run this server only for handling HTTP ACME challange
		go http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if myACME.HandleHTTPChallenge(w, r) {
				return
			}
		}),
		)

		err := magic.ManageAsync(context.Background(), []string{domain})
		if err != nil {
			return fmt.Errorf("manage sync: %w", err)
		}

		config := &tls.Config{
			NextProtos: []string{"http/1.1", http2.NextProtoTLS, "h2-14"}, // h2-14 is just for compatibility. will be eventually removed.
		}
		config.GetCertificate = magic.GetCertificate
		config.NextProtos = append(config.NextProtos, tlsalpn01.ACMETLS1Protocol)

		lis = tls.NewListener(lis, config)
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

	gwmux := runtime.NewServeMux()
	if err := pb.RegisterTaxiHandlerFromEndpoint(
		context.Background(),
		gwmux, address, []g.DialOption{g.WithInsecure()},
	); err != nil {
		return err
	}

	go grpcServer.Serve(grpcL)
	go http.Serve(httpL, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if isGrpcWebRequest(req) {
			grpcWebServer.ServeHTTP(resp, req)
		} else {
			gwmux.ServeHTTP(resp, req)
		}
	}))

	go mux.Serve()
	return nil
}

func isGrpcWebRequest(req *http.Request) bool {
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
