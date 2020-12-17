package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/application"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
	"github.com/vulpemventures/liquid-taxi/internal/interfaces/grpc"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
	g "google.golang.org/grpc"
)

func main() {
	log.Info("Taxi server is starting...")

	topupRepository := inmemory.NewInmemoryTopupRepository()
	lockRepository := inmemory.NewInmemoryLockRepository()
	priceRepository := inmemory.NewInmemoryPriceRepository()

	priceRepository.UpdateOrCreatePrice(context.Background(), domain.Price{
		AssetHash: config.GetAssetHash(),
		Amount:    decimal.NewFromFloat(10.154),
	})

	taxiSvcOptions := application.NewTaxiServiceArgs{
		ExplorerEndpoint: config.GetExplorerEndpoint(),
		Mnemonic:         strings.Split(config.GetMnemonic(), " "),
		Network:          *config.GetNetwork(),
		FixedFeeAmount:   uint64(config.GetFixedFeeAmount()),
		LockRepository:   lockRepository,
		PriceRepository:  priceRepository,
		TopupRepository:  topupRepository,
	}

	taxiSvc, err := application.NewTaxiService(taxiSvcOptions)
	if err != nil {
		log.Fatal("taxi service creation error ", err)
	}

	adminSvc := application.NewAdminService(taxiSvc)

	taxiHandler := grpc.NewTaxiHandler(taxiSvc)
	adminHandler := grpc.NewAdminHandler(adminSvc)

	taxiServer := g.NewServer()
	adminServer := g.NewServer()

	pb.RegisterAdminServer(adminServer, adminHandler)
	pb.RegisterTaxiServer(taxiServer, taxiHandler)

	domainURL := "localhost"
	taxiAddress := fmt.Sprintf("%s:%+v", domainURL, config.GetTaxiPort())
	adminAddress := fmt.Sprintf("%s:%+v", domainURL, config.GetAdminPort())

	// serve
	lisTaxi, err := net.Listen("tcp", taxiAddress)
	if err != nil {
		log.Fatal("failed to listen ", err)
	}

	lisAdmin, err := net.Listen("tcp", adminAddress)
	if err != nil {
		log.Fatal("failed to listen ", err)
	}

	go func() {
		err := taxiServer.Serve(lisTaxi)
		if err != nil {
			log.Fatal("failed to serve ", err)
		}
	}()

	log.Info("taxi server is listening on ", taxiAddress)

	go func() {
		err := adminServer.Serve(lisAdmin)
		if err != nil {
			log.Fatal("failed to serve ", err)
		}
	}()

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
