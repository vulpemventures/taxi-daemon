package application

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

func TestBalance(t *testing.T) {
	hdTreeRepo := inmemory.NewInmemoryHDTreeRepository()
	topupRepo := newMockedTopupRepository([]string{superToken, hyperToken})
	priceRepo := newMockedPriceRepository(nil)
	explorerSvc := explorer.NewService(explorerEndpoint)

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc)

	balanceInfosBefore, err := adminSvc.Balance(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	payoutAddresses, err := hdTreeRepo.GetNewPayoutAddresses(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = explorerSvc.Mint(payoutAddresses[0], 100)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

	balanceInfosAfter, err := adminSvc.Balance(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, len(balanceInfosAfter)-len(balanceInfosBefore) >= 1)
}

func TestListTopups(t *testing.T) {
	hdTreeRepo := inmemory.NewInmemoryHDTreeRepository()
	topupRepo := newMockedTopupRepository([]string{superToken, hyperToken})
	priceRepo := newMockedPriceRepository(nil)
	explorerSvc := explorer.NewService(explorerEndpoint)

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc)

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
	explorerSvc := explorer.NewService(explorerEndpoint)

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc)
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
	explorerSvc := explorer.NewService(explorerEndpoint)

	config.Set(config.MnemonicKey, mnemonic)
	t.Cleanup(func() {
		config.Set(config.MnemonicKey, "")
	})

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc)
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
	explorerSvc := explorer.NewService(explorerEndpoint)

	config.Set(config.MnemonicKey, mnemonic)
	t.Cleanup(func() {
		config.Set(config.MnemonicKey, "")
	})

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc)
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
	explorerSvc := explorer.NewService(explorerEndpoint)

	config.Set(config.MnemonicKey, mnemonic)
	t.Cleanup(func() {
		config.Set(config.MnemonicKey, "")
	})

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc)
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

func TestSendToMany(t *testing.T) {
	config.Set(config.MnemonicKey, mnemonic)
	t.Cleanup(func() {
		config.Set(config.MnemonicKey, "")
	})

	output := wallet.Receiver{
		Address: "XP3vdsDLz5DojUjAqkLPqcajFtz5wzS73B",
		Asset:   regtest.AssetID,
		Amount:  100,
	}

	hdTreeRepo := newMockedHDTreeRepository(10)
	topupRepo := newMockedTopupRepository(nil)
	priceRepo := newMockedPriceRepository(nil)
	explorerSvc := explorer.NewService(explorerEndpoint)

	addrs, err := hdTreeRepo.GetNewFundingAddresses(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}

	_, err = faucet(addrs[1], explorerEndpoint)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc)

	_, err = adminSvc.SendToMany(context.Background(), []wallet.Receiver{output}, 1000, true)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	utxos, err := explorerSvc.GetUnspents(output.Address, [][]byte{})
	if err != nil {
		t.Fatal(err)
	}

	fundsReceived := false
	for _, utxo := range utxos {
		if utxo.Asset() == regtest.AssetID && utxo.Value() == output.Amount {
			fundsReceived = true
		}
	}

	assert.Equal(t, true, fundsReceived)
}
