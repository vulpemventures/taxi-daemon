package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/taxi-daemon/internal/config"
	"github.com/vulpemventures/taxi-daemon/internal/core/application"
	oceanwallet "github.com/vulpemventures/taxi-daemon/internal/infrastructure/ocean-wallet"
	dbbadger "github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/badger"
	"github.com/vulpemventures/taxi-daemon/internal/interfaces/grpc"
)

var (
	datadir         = config.GetString(config.DatadirKey)
	oceanWalletAddr = config.GetString(config.OceanWalletAddrKey)
	logLevel        = config.GetInt(config.LogLevelKey)

	adminPort        = config.GetInt(config.AdminPortKey)
	taxiPort         = config.GetInt(config.TaxiPortKey)
	tlsKeyPath       = config.GetString(config.TLSKeyPathKey)
	tlsCertPath      = config.GetString(config.TLSCertPathKey)
	email            = config.GetString(config.EmailKey)
	domain           = config.GetString(config.DomainKey)
	adminExternalURL = config.GetString(config.AdminExternalUrlKey)
	taxiExternalURL  = config.GetString(config.TaxiExternalUrlKey)
)

func main() {
	log.SetLevel(log.Level(logLevel))

	oceanWallet, err := oceanwallet.NewInsecureOceanWallet(oceanWalletAddr)
	if err != nil {
		log.Fatalf(
			"failed to connect to ocean wallet at addr %s: %s", oceanWalletAddr, err,
		)
	}
	defer oceanWallet.Close()

	wallet, err := application.NewWallet(oceanWallet)
	if err != nil {
		log.Fatal(err)
	}

	repoManager, err := dbbadger.NewRepoManager(datadir, log.New())
	if err != nil {
		log.Fatalf("failed to open db: %s", err)
	}
	defer repoManager.Close()

	adminSvc := application.NewAdminService(wallet, repoManager)
	taxiSvc := application.NewTaxiService(wallet, repoManager)

	service, err := grpc.NewService(grpc.ServiceOpts{
		AdminSvc:         adminSvc,
		TaxiSvc:          taxiSvc,
		AdminPort:        adminPort,
		TaxiPort:         taxiPort,
		TLSKeyPath:       tlsKeyPath,
		TLSCertPath:      tlsCertPath,
		Email:            email,
		Domain:           domain,
		AdminExternalURL: adminExternalURL,
		TaxiExternalURL:  taxiExternalURL,
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
