package application

import (
	"context"
	"testing"
	"time"

	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/tj/assert"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
)

func TestListTopups(t *testing.T) {
	newTaxiOpts := NewTaxiServiceArgs{
		Mnemonic:        words,
		Network:         regtest,
		FixedFeeAmount:  100000000,
		TopupRepository: inmemory.NewInmemoryTopupRepository(),
		LockRepository:  inmemory.NewInmemoryLockRepository(),
		PriceRepository: newMockedPriceRepository(t, []string{superToken, hyperToken}),
		ExplorerSvc:     explorer.NewService(explorerEndpoint),
	}

	taxiSvc, err := newTaxiService(newTaxiOpts)
	if err != nil {
		t.Fatal(err)
	}

	// found taxi with LBTC
	addresses, err := taxiSvc.getAddresses()
	addr := addresses[0]
	if err != nil {
		t.Fatal(err)
	}

	_, err = faucet(addr, explorerEndpoint)
	if err != nil {
		t.Fatal(nil)
	}
	time.Sleep(5 * time.Second)

	// do some topups
	_, err = taxiSvc.TopupWithAsset(context.Background(), superToken)
	if err != nil {
		t.Fatal(err)
	}

	topupHyperToken, err := taxiSvc.TopupWithAsset(context.Background(), hyperToken)
	if err != nil {
		t.Fatal(err)
	}

	adminSvc := NewAdminService(taxiSvc.topupRepository)
	topups, err := adminSvc.ListTopups(context.Background(), "")
	assert.Nil(t, err)
	assert.NotNil(t, topups)
	assert.Equal(t, 2, len(topups))

	filteredTopups, err := adminSvc.ListTopups(context.Background(), hyperToken)
	assert.Nil(t, err)
	assert.NotNil(t, filteredTopups)
	assert.Equal(t, []domain.Topup{*topupHyperToken}, filteredTopups)
}
