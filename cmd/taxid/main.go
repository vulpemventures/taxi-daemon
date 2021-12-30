package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	explorer "github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/vulpemventures/taxi-daemon/config"
	"github.com/vulpemventures/taxi-daemon/internal/application"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/dbbadger"
	"github.com/vulpemventures/taxi-daemon/internal/interfaces/grpc"
)

var (
	datadirPath    = config.GetString(config.DatadirKey)
	net            = config.GetNetwork()
	mnemonic       = config.GetMnemonic()
	utxoTTL        = time.Duration(config.GetInt(config.UtxoTTLKey)) * time.Second
	cacheInterval  = time.Duration(config.GetInt(config.UtxoCacheIntervalKey)) * time.Second
	fixedFeeAmount = uint64(config.GetInt(config.FixedFeeAmountKey))

	adminPort   = config.GetInt(config.AdminPortKey)
	taxiPort    = config.GetInt(config.TaxiPortKey)
	tlsKeyPath  = config.GetString(config.TLSKeyPathKey)
	tlsCertPath = config.GetString(config.TLSCertPathKey)
	email       = config.GetString(config.EmailKey)
	domain      = config.GetString(config.DomainKey)
)

func main() {
	log.Info("---- Liquid Taxi ----")

	log.Info("loading datadir... ", datadirPath)
	err := dbbadger.CreateDirIfNotExist(datadirPath)
	if err != nil {
		log.Fatal(err)
	}

	dbLogger := log.New()
	hdTreeRepository, err := dbbadger.NewBadgerHDTreeRepository(datadirPath, mnemonic, *net, dbLogger)
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

	cache := application.NewCache(application.NewCacheArgs{
		Explorer:         explorerSvc,
		HdTreeRepository: hdTreeRepository,
		LockRepository:   lockRepository,
		FragmentAmount:   fixedFeeAmount,
		FragmentAsset:    net.AssetID,
		Interval:         cacheInterval,
	})

	argsTaxiSvc := application.NewTaxiServiceArgs{
		HdTreeRepo:     hdTreeRepository,
		LockRepo:       lockRepository,
		TopupRepo:      topupRepository,
		PriceRepo:      priceRepository,
		Network:        *net,
		FixedFeeAmount: fixedFeeAmount,
		Mnemonic:       mnemonic,
		Cache:          cache,
	}
	taxiSvc := application.NewTaxiService(argsTaxiSvc)

	adminSvcArgs := application.NewAdminServiceArgs{
		HDTreeRepository: hdTreeRepository,
		TopupRepository:  topupRepository,
		PriceRepository:  priceRepository,
		LockRepository:   lockRepository,
		ExplorerService:  explorerSvc,
		Mnemonic:         mnemonic,
		Network:          *net,
		Cache:            cache,
	}
	adminSvc := application.NewAdminService(adminSvcArgs)

	service, err := grpc.NewService(grpc.ServiceOpts{
		AdminSvc: adminSvc,
		TaxiSvc:  taxiSvc,
		AdminConfig: grpc.ServiceConfig{
			Port:        adminPort,
			TLSKeyPath:  tlsKeyPath,
			TLSCertPath: tlsCertPath,
			Email:       email,
			Domain:      domain,
		},
		TaxiConfig: grpc.ServiceConfig{
			Port:        taxiPort,
			TLSKeyPath:  tlsKeyPath,
			TLSCertPath: tlsCertPath,
			Email:       email,
			Domain:      domain,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := service.Start(); err != nil {
		log.Fatal(err)
	}
	defer service.Stop()
	// Interrupt Notification.
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-interrupt
	log.Info("shutting down Taxi service...")
}
