package application

import (
	"testing"
	"time"

	"github.com/tj/assert"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
)

func TestListTopups(t *testing.T) {
	newTaxiOpts := NewTaxiServiceArgs{
		ExplorerEndpoint: explorerEndpoint,
		Mnemonic:         words,
		Network:          regtest,
		FixedFeeAmount:   100000000,
		TopupRepository:  inmemory.NewInmemoryTopupRepository(),
		LockRepository:   inmemory.NewInmemoryLockRepository(),
		PriceRepository:  newMockedPriceRepository(t, []string{superToken, hyperToken}),
	}

	taxiSvc, err := newTaxiService(newTaxiOpts)
	if err != nil {
		t.Error(err)
	}

	// found taxi with LBTC
	addr, _, err := taxiSvc.getAddresses()
	if err != nil {
		t.Error(err)
	}

	_, err = faucet(addr, explorerEndpoint)
	if err != nil {
		t.Error(nil)
	}
	time.Sleep(5 * time.Second)

	// do some topups
	_, err = taxiSvc.TopupWithAsset(superToken)
	if err != nil {
		t.Error(err)
	}

	topupHyperToken, err := taxiSvc.TopupWithAsset(hyperToken)
	if err != nil {
		t.Error(err)
	}

	adminSvc := NewAdminService(taxiSvc)
	topups, err := adminSvc.ListTopups("")
	assert.Nil(t, err)
	assert.NotNil(t, topups)
	assert.Equal(t, 2, len(topups))

	filteredTopups, err := adminSvc.ListTopups(hyperToken)
	assert.Nil(t, err)
	assert.NotNil(t, filteredTopups)
	assert.Equal(t, []domain.Topup{*topupHyperToken}, filteredTopups)
}
