package main

import (
	"fmt"
	"net"
	"strings"

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

	taxiSvcOptions := application.NewTaxiServiceArgs{
		ExplorerEndpoint: config.GetExplorerEndpoint(),
		Mnemonic:         strings.Split(config.GetMnemonic(), " "),
		Network:          *config.GetNetwork(),
		AmountToConvert:  uint64(config.GetFragmentAmount()),
		InitialPrices: []domain.Price{
			// TODO this should be configurable from config.json file for intance
			domain.Price{
				AssetHash: "2e4c430667f861d010b6c78ce6a3943d35d84ca77ca1898031b8799c4798bc4f",
				Amount:    decimal.NewFromFloat(10.894),
			},
		},
		TopupRepository: topupRepository,
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

	domainUrl := "localhost"
	taxiAddress := fmt.Sprintf("%s:%+v", domainUrl, config.GetTaxiPort())
	adminAddress := fmt.Sprintf("%s:%+v", domainUrl, config.GetAdminPort())

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
}
