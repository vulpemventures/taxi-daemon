package application

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
)

func TestListTopups(t *testing.T) {
	hdTreeRepo := inmemory.NewInmemoryHDTreeRepository()
	topupRepo := newMockedTopupRepository([]string{superToken, hyperToken})
	priceRepo := newMockedPriceRepository(nil)

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo)

	topups, err := adminSvc.ListTopups(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(topups))

	filteredTopups, err := adminSvc.ListTopups(context.Background(), hyperToken)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 1, len(filteredTopups))
}

func TestAddAsset(t *testing.T) {
	hdTreeRepo := inmemory.NewInmemoryHDTreeRepository()
	topupRepo := newMockedTopupRepository(nil)
	priceRepo := newMockedPriceRepository(nil)

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo)
	ctx := context.Background()

	// add asset
	if err := adminSvc.AddAsset(
		ctx,
		hyperToken,
		decimal.NewFromFloat(27169),
	); err != nil {
		t.Fatal(err)
	}

	all, err := priceRepo.GetAllPrices(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(all))

	// update same asset
	if err := adminSvc.AddAsset(
		ctx,
		hyperToken,
		decimal.NewFromFloat(27252),
	); err != nil {
		t.Fatal(err)
	}

	all, err = priceRepo.GetAllPrices(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(all))
}

func TestGetFundingAddresses(t *testing.T) {
	hdTreeRepo := newMockedHDTreeRepository(0)
	topupRepo := newMockedTopupRepository(nil)
	priceRepo := newMockedPriceRepository(nil)

	config.Set(config.MnemonicKey, mnemonic)
	t.Cleanup(func() {
		config.Set(config.MnemonicKey, "")
	})

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo)
	ctx := context.Background()

	addresses, err := adminSvc.GetFundingAddresses(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(addresses))
}

func TestLatestFundingAddressIndex(t *testing.T) {
	hdTreeRepo := newMockedHDTreeRepository(3)
	topupRepo := newMockedTopupRepository(nil)
	priceRepo := newMockedPriceRepository(nil)

	config.Set(config.MnemonicKey, mnemonic)
	t.Cleanup(func() {
		config.Set(config.MnemonicKey, "")
	})

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo)
	ctx := context.Background()

	lastIndex, err := adminSvc.LatestFundingAddressIndex(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, lastIndex)
}

func TestListFundingAddresses(t *testing.T) {
	hdTreeRepo := newMockedHDTreeRepository(10)
	topupRepo := newMockedTopupRepository(nil)
	priceRepo := newMockedPriceRepository(nil)

	config.Set(config.MnemonicKey, mnemonic)
	t.Cleanup(func() {
		config.Set(config.MnemonicKey, "")
	})

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo)
	ctx := context.Background()

	addresses, err := adminSvc.ListFundingAddresses(ctx, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 10, len(addresses))

	addresses, err = adminSvc.ListFundingAddresses(ctx, 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 5, len(addresses))
}
