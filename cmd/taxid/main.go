package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/application"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/dbbadger"
	"github.com/vulpemventures/liquid-taxi/internal/interfaces/grpc"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
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

	log.Info("Taxi server is starting...")

	taxiSvcOptions := application.NewTaxiServiceArgs{
		Mnemonic:        strings.Split(config.GetString(config.MnemonicKey), " "),
		Network:         *config.GetNetwork(),
		FixedFeeAmount:  uint64(config.GetInt(config.FixedFeeAmountKey)),
		LockRepository:  lockRepository,
		PriceRepository: priceRepository,
		TopupRepository: topupRepository,
		ExplorerSvc:     explorer.NewService(config.GetString(config.ExplorerEndpointKey)),
	}

	taxiSvc, err := application.NewTaxiService(taxiSvcOptions)
	if err != nil {
		log.Fatal("taxi service creation error ", err)
	}

	adminSvc := application.NewAdminService(
		topupRepository,
		priceRepository,
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
