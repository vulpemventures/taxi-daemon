package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	explorer "github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/vulpemventures/taxi-daemon/internal/application"
	"github.com/vulpemventures/taxi-daemon/internal/domain"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/inmemory"
	"github.com/vulpemventures/taxi-daemon/pkg/wallet"
)

func TestListLocks(t *testing.T) {
	hdTreeRepo := newMockedHDTreeRepository(0)
	topupRepo := newMockedTopupRepository([]string{superToken, hyperToken})
	priceRepo := newMockedPriceRepository(nil)
	lockRepo := inmemory.NewInmemoryLockRepository(utxoTTL)
	explorerSvc, err := explorer.NewService(explorerEndpoint)
	if err != nil {
		panic(err)
	}
	cache := application.NewCache(application.NewCacheArgs{
		Explorer:         explorerSvc,
		HdTreeRepository: hdTreeRepo,
		LockRepository:   lockRepo,
		FragmentAsset:    regtest.AssetID,
		FragmentAmount:   fixedFeeAmount,
		Interval:         30 * time.Second,
	})
	newAdminSvcArgs := application.NewAdminServiceArgs{
		HDTreeRepository: hdTreeRepo,
		TopupRepository:  topupRepo,
		PriceRepository:  priceRepo,
		LockRepository:   lockRepo,
		ExplorerService:  explorerSvc,
		Mnemonic:         words,
		Network:          regtest,
		Cache:            cache,
	}
	adminSvc := application.NewAdminService(newAdminSvcArgs)

	lockRepo.Lock(context.Background(), []domain.Lock{
		domain.Lock{
			TxID: "123456789",
			Vout: 1,
		},
		domain.Lock{
			TxID: "987654321",
			Vout: 666,
		},
	})

	locks, err := adminSvc.ListLocks(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(locks))
}

func TestBalance(t *testing.T) {
	hdTreeRepo := newMockedHDTreeRepository(0)
	topupRepo := newMockedTopupRepository([]string{superToken, hyperToken})
	priceRepo := newMockedPriceRepository(nil)
	lockRepo := inmemory.NewInmemoryLockRepository(utxoTTL)
	explorerSvc, err := explorer.NewService(explorerEndpoint)
	if err != nil {
		panic(err)
	}
	cache := application.NewCache(application.NewCacheArgs{
		Explorer:         explorerSvc,
		HdTreeRepository: hdTreeRepo,
		LockRepository:   lockRepo,
		FragmentAsset:    regtest.AssetID,
		FragmentAmount:   fixedFeeAmount,
		Interval:         30 * time.Second,
	})
	newAdminSvcArgs := application.NewAdminServiceArgs{
		HDTreeRepository: hdTreeRepo,
		TopupRepository:  topupRepo,
		PriceRepository:  priceRepo,
		LockRepository:   lockRepo,
		ExplorerService:  explorerSvc,
		Mnemonic:         words,
		Network:          regtest,
		Cache:            cache,
	}
	adminSvc := application.NewAdminService(newAdminSvcArgs)

	balanceInfosBefore, err := adminSvc.Balance(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	payoutAddresses, err := hdTreeRepo.GetNewPayoutAddresses(context.Background(), 1, false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = explorerSvc.Faucet(payoutAddresses[0])
	if err != nil {
		t.Fatal(err)
	}

	_, err = explorerSvc.Faucet(payoutAddresses[0])
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

	assert.Equal(t, true, len(balanceInfosAfter)-len(balanceInfosBefore) >= 2)
}

func TestListTopups(t *testing.T) {
	adminSvc := newMockedAdminService(0)

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
	hdTreeRepo := newMockedHDTreeRepository(0)
	topupRepo := newMockedTopupRepository([]string{superToken, hyperToken})
	priceRepo := newMockedPriceRepository(nil)
	lockRepo := inmemory.NewInmemoryLockRepository(utxoTTL)
	explorerSvc, err := explorer.NewService(explorerEndpoint)
	if err != nil {
		panic(err)
	}
	cache := application.NewCache(application.NewCacheArgs{
		Explorer:         explorerSvc,
		HdTreeRepository: hdTreeRepo,
		LockRepository:   lockRepo,
		FragmentAsset:    regtest.AssetID,
		FragmentAmount:   fixedFeeAmount,
		Interval:         30 * time.Second,
	})
	newAdminSvcArgs := application.NewAdminServiceArgs{
		HDTreeRepository: hdTreeRepo,
		TopupRepository:  topupRepo,
		PriceRepository:  priceRepo,
		LockRepository:   lockRepo,
		ExplorerService:  explorerSvc,
		Mnemonic:         words,
		Network:          regtest,
		Cache:            cache,
	}
	adminSvc := application.NewAdminService(newAdminSvcArgs)

	ctx := context.Background()

	// add asset
	if err := adminSvc.AddAsset(
		ctx,
		hyperToken,
		decimal.NewFromFloat(27169),
		// take 20% as service fee
		200,
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
		200,
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
	adminSvc := newMockedAdminService(0)

	ctx := context.Background()

	addresses, err := adminSvc.GetFundingAddresses(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(addresses))
}

func TestLatestFundingAddressIndex(t *testing.T) {
	adminSvc := newMockedAdminService(3)

	ctx := context.Background()

	lastIndex, err := adminSvc.LatestFundingAddressIndex(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, lastIndex)
}

func TestListFundingAddresses(t *testing.T) {
	adminSvc := newMockedAdminService(10)

	ctx := context.Background()

	addresses, err := adminSvc.ListFundingAddresses(ctx, 0, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 10, len(addresses))

	addresses, err = adminSvc.ListFundingAddresses(ctx, 1, 5, false)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 5, len(addresses))
}

func TestSendToMany(t *testing.T) {
	hdTreeRepo := newMockedHDTreeRepository(0)
	topupRepo := newMockedTopupRepository([]string{superToken, hyperToken})
	priceRepo := newMockedPriceRepository(nil)
	lockRepo := inmemory.NewInmemoryLockRepository(utxoTTL)
	explorerSvc, err := explorer.NewService(explorerEndpoint)
	if err != nil {
		panic(err)
	}
	cache := application.NewCache(application.NewCacheArgs{
		Explorer:         explorerSvc,
		HdTreeRepository: hdTreeRepo,
		LockRepository:   lockRepo,
		FragmentAsset:    regtest.AssetID,
		FragmentAmount:   fixedFeeAmount,
		Interval:         30 * time.Second,
	})
	newAdminSvcArgs := application.NewAdminServiceArgs{
		HDTreeRepository: hdTreeRepo,
		TopupRepository:  topupRepo,
		PriceRepository:  priceRepo,
		LockRepository:   lockRepo,
		ExplorerService:  explorerSvc,
		Mnemonic:         words,
		Network:          regtest,
		Cache:            cache,
	}
	adminSvc := application.NewAdminService(newAdminSvcArgs)

	addrs, err := hdTreeRepo.GetNewFundingAddresses(context.Background(), 2, false)
	if err != nil {
		t.Fatal(err)
	}

	payoutAddrs, err := hdTreeRepo.GetNewPayoutAddresses(context.Background(), 1, false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = explorerSvc.Faucet(addrs[1])
	if err != nil {
		t.Fatal(err)
	}

	_, shitcoin, err := explorerSvc.Mint(payoutAddrs[0], 1000000)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

	output := wallet.Receiver{
		Address: "XP3vdsDLz5DojUjAqkLPqcajFtz5wzS73B",
		Asset:   regtest.AssetID,
		Amount:  100000,
	}

	outputBis := wallet.Receiver{
		Address: "XP3vdsDLz5DojUjAqkLPqcajFtz5wzS73B",
		Asset:   shitcoin,
		Amount:  32000,
	}

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
