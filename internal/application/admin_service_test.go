package application

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/tj/assert"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
)

func TestListTopups(t *testing.T) {
	priceRepo := newMockedPriceRepository(t, []string{superToken, hyperToken})

	newTaxiOpts := NewTaxiServiceArgs{
		Mnemonic:        words,
		Network:         regtest,
		FixedFeeAmount:  100000000,
		TopupRepository: inmemory.NewInmemoryTopupRepository(),
		LockRepository:  inmemory.NewInmemoryLockRepository(),
		PriceRepository: priceRepo,
		ExplorerSvc:     explorer.NewService(explorerEndpoint),
		UtxoTTL:         time.Minute,
	}

	taxiSvc, err := newTaxiService(newTaxiOpts)
	if err != nil {
		t.Fatal(err)
	}

	// found taxi with LBTC
	addr, err := taxiSvc.getFundAddress()
	if err != nil {
		t.Fatal(err)
	}

	_, err = faucet(addr, explorerEndpoint)
	if err != nil {
		t.Fatal(nil)
	}

	_, err = faucet(addr, explorerEndpoint)
	if err != nil {
		t.Fatal(nil)
	}

	// do some topups
	_, err = taxiSvc.TopupWithAsset(context.Background(), superToken)
	if err != nil {
		t.Fatal(err)
	}

	topupHyperToken, err := taxiSvc.TopupWithAsset(context.Background(), hyperToken)
	if err != nil {
		t.Fatal(err)
	}

	adminSvc := NewAdminService(taxiSvc.topupRepository, priceRepo)
	topups, err := adminSvc.ListTopups(context.Background(), "")
	assert.Nil(t, err)
	assert.NotNil(t, topups)
	assert.Equal(t, 2, len(topups))

	filteredTopups, err := adminSvc.ListTopups(context.Background(), hyperToken)
	assert.Nil(t, err)
	assert.NotNil(t, filteredTopups)
	assert.Equal(t, []domain.Topup{*topupHyperToken}, filteredTopups)
}

func TestAddAsset(t *testing.T) {
	// init repo with 0 prices
	priceRepo := newMockedPriceRepository(t, []string{})

	adminSvc := NewAdminService(
		inmemory.NewInmemoryTopupRepository(),
		priceRepo,
	)

	price666 := decimal.NewFromFloat(666)

	err := adminSvc.AddAsset(context.Background(), hyperToken, price666)
	if err != nil {
		t.Fatal(err)
	}

	all, err := priceRepo.GetAllPrices(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(all))
	assert.Equal(t, true, price666.Equal(all[0].Amount))

	price777 := decimal.NewFromFloat(777)

	err = adminSvc.AddAsset(context.Background(), hyperToken, price777)
	if err != nil {
		t.Fatal(err)
	}

	all, err = priceRepo.GetAllPrices(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(all))
	assert.Equal(t, true, price777.Equal(all[0].Amount))
}
