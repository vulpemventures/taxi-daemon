package application

import (
	"context"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	explorer "github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/taxi-daemon/internal/domain"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/inmemory"
)

const (
	mnemonic         = "clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act soccer drive year mask"
	explorerEndpoint = "http://localhost:3001"
	superToken       = "2e4c430667f861d010b6c78ce6a3943d35d84ca77ca1898031b8799c4798bc4f"
	hyperToken       = "6666430667f861d010b6c78ce6a3943d35d84ca77ca1898031b8799c47986666"
	fixedFeeAmount   = 1000
)

var (
	regtest             = network.Regtest
	words               = strings.Split(mnemonic, " ")
	defaultSatsPerByte  = decimal.NewFromFloat(0.1)
	doesnotmatterAmount = decimal.NewFromFloat(66.6)
	utxoTTL             = time.Minute
)

func newMockedPriceRepository(assets []string) domain.PriceRepository {
	repo := inmemory.NewInmemoryPriceRepository()
	initialPrices := []domain.Price{}
	ctx := context.Background()

	for _, a := range assets {
		initialPrices = append(initialPrices, domain.Price{
			AssetHash: a,
			Amount:    doesnotmatterAmount,
		})
	}

	for _, price := range initialPrices {
		repo.UpdateOrCreatePrice(ctx, price)
	}

	return repo
}

func newMockedTopupRepository(assets []string) domain.TopupRepository {
	fee := uint64(1000)
	price := decimal.NewFromInt(25000)
	expirationTime := time.Now().Add(10 * time.Minute)

	repo := inmemory.NewInmemoryTopupRepository()
	topups := make([]domain.Topup, len(assets), len(assets))
	for i, asset := range assets {
		topups[i] = domain.NewTopup(asset, fee, price, expirationTime)
	}

	ctx := context.Background()
	for _, t := range topups {
		repo.AddTopup(ctx, t)
	}

	return repo
}

func newMockedHDTreeRepository(numOfAddresses uint) domain.HDTreeRepository {
	repo := inmemory.NewInmemoryHDTreeRepository(words, regtest)
	ctx := context.Background()

	repo.GetOrCreateHDTree(ctx)
	if numOfAddresses > 0 {
		repo.GetNewFundingAddresses(ctx, numOfAddresses, false)
		repo.GetNewPayoutAddresses(ctx, numOfAddresses, false)
	}

	return repo
}

func newMockedLockRepository() domain.LockRepository {
	return inmemory.NewInmemoryLockRepository(utxoTTL)
}

func newMockedAdminService(numberOfAddresses uint) *adminService {
	hdTreeRepo := newMockedHDTreeRepository(numberOfAddresses)
	topupRepo := newMockedTopupRepository([]string{superToken, hyperToken})
	priceRepo := newMockedPriceRepository(nil)
	lockRepo := inmemory.NewInmemoryLockRepository(utxoTTL)
	explorerSvc, err := explorer.NewService(explorerEndpoint)
	if err != nil {
		panic(err)
	}

	cache := newCache(NewCacheArgs{
		Explorer:         explorerSvc,
		HdTreeRepository: hdTreeRepo,
		LockRepository:   lockRepo,
		FragmentAsset:    regtest.AssetID,
		FragmentAmount:   fixedFeeAmount,
		Interval:         30 * time.Second,
	})

	newAdminSvcArgs := NewAdminServiceArgs{
		HDTreeRepository: hdTreeRepo,
		TopupRepository:  topupRepo,
		PriceRepository:  priceRepo,
		LockRepository:   lockRepo,
		ExplorerService:  explorerSvc,
		Mnemonic:         words,
		Network:          regtest,
		Cache:            cache,
	}

	return newAdminService(newAdminSvcArgs)
}
