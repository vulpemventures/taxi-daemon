package application

import (
	"context"
	"testing"
	"time"

	"github.com/tj/assert"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
	"github.com/vulpemventures/liquid-taxi/pkg/explorer"
)

func TestListTopups(t *testing.T) {
	newTaxiOpts := NewTaxiServiceArgs{
		Mnemonic:        words,
		Network:         regtest,
		FixedFeeAmount:  100000000,
		TopupRepository: inmemory.NewInmemoryTopupRepository(),
		LockRepository:  inmemory.NewInmemoryLockRepository(),
		PriceRepository: newMockedPriceRepository(t, []string{superToken, hyperToken}),
		ExplorerSvc:     explorer.NewExplorerService(explorerEndpoint),
	}

	taxiSvc, err := newTaxiService(newTaxiOpts)
	if err != nil {
		t.Error(err)
	}

	// found taxi with LBTC
	addresses, err := taxiSvc.getAddresses()
	addr := addresses[0]
	if err != nil {
		t.Error(err)
	}

	_, err = faucet(addr, explorerEndpoint)
	if err != nil {
		t.Error(nil)
	}
	time.Sleep(5 * time.Second)

	// do some topups
	_, err = taxiSvc.TopupWithAsset(context.Background(), superToken)
	if err != nil {
		t.Error(err)
	}

	topupHyperToken, err := taxiSvc.TopupWithAsset(context.Background(), hyperToken)
	if err != nil {
		t.Error(err)
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
