package application

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

func TestBalance(t *testing.T) {
	hdTreeRepo := inmemory.NewInmemoryHDTreeRepository()
	topupRepo := newMockedTopupRepository([]string{superToken, hyperToken})
	priceRepo := newMockedPriceRepository(nil)
	explorerSvc := explorer.NewService(explorerEndpoint)

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc, words, regtest)

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

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc, words, regtest)

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

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc, words, regtest)
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

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc, words, regtest)
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

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc, words, regtest)
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

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc, words, regtest)
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
	hdTreeRepo := newMockedHDTreeRepository(10)
	topupRepo := newMockedTopupRepository(nil)
	priceRepo := newMockedPriceRepository(nil)
	explorerSvc := explorer.NewService(explorerEndpoint)

	addrs, err := hdTreeRepo.GetNewFundingAddresses(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}

	payoutAddrs, err := hdTreeRepo.GetNewPayoutAddresses(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = explorerSvc.Faucet(addrs[1])
	if err != nil {
		t.Fatal(err)
	}

	_, shitcoin, err := explorerSvc.Mint(payoutAddrs[0], 100)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

	output := wallet.Receiver{
		Address: "XP3vdsDLz5DojUjAqkLPqcajFtz5wzS73B",
		Asset:   regtest.AssetID,
		Amount:  100,
	}

	outputBis := wallet.Receiver{
		Address: "XP3vdsDLz5DojUjAqkLPqcajFtz5wzS73B",
		Asset:   shitcoin,
		Amount:  32,
	}

	adminSvc := newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc, words, regtest)

	_, err = adminSvc.SendToMany(context.Background(), []wallet.Receiver{output, outputBis}, 1000, true)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	utxos, err := explorerSvc.GetUnspents(output.Address, [][]byte{})
	if err != nil {
		t.Fatal(err)
	}

	btcReceived, shitcoinReceived := false, false

	for _, utxo := range utxos {
		if utxo.Asset() == output.Asset && utxo.Value() == output.Amount {
			btcReceived = true
		}

		if utxo.Asset() == outputBis.Asset && utxo.Value() == outputBis.Amount {
			shitcoinReceived = true
		}
	}

	assert.Equal(t, true, btcReceived)
	assert.Equal(t, true, shitcoinReceived)
}
