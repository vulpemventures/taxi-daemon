package application

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tj/assert"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
)

func TestListTopups(t *testing.T) {
	newTaxiOpts := NewTaxiServiceArgs{
		ExplorerEndpoint: explorerEndpoint,
		Mnemonic:         words,
		Network:          regtest,
		BasisPoint:       10,
		TopupRepository:  inmemory.NewInmemoryTopupRepository(),
		InitialPrices: []domain.Price{
			domain.Price{
				AssetHash: superToken,
				Amount:    decimal.NewFromFloat(8.0026),
			},
			domain.Price{
				AssetHash: hyperToken,
				Amount:    decimal.NewFromFloat(7.032),
			},
		},
	}

	taxiSvc, err := NewTaxiService(newTaxiOpts)
	if err != nil {
		t.Error(err)
	}

	// found taxi with LBTC
	addr, _ := taxiSvc.getAddresses()
	_, err = faucet(addr, explorerEndpoint)
	if err != nil {
		t.Error(nil)
	}
	time.Sleep(5 * time.Second)

	// do some topups
	_, err = taxiSvc.TopupWithAsset(superToken, 5000, defaultSatsPerByte)
	if err != nil {
		t.Error(err)
	}

	topupHyperToken, err := taxiSvc.TopupWithAsset(hyperToken, 10000, defaultSatsPerByte)
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

func TestChangeSpread(t *testing.T) {
	newTaxiOpts := NewTaxiServiceArgs{
		ExplorerEndpoint: explorerEndpoint,
		Mnemonic:         words,
		Network:          regtest,
		BasisPoint:       10,
		TopupRepository:  inmemory.NewInmemoryTopupRepository(),
		InitialPrices: []domain.Price{
			domain.Price{
				AssetHash: superToken,
				Amount:    decimal.NewFromFloat(8.0026),
			},
		},
	}

	taxiSvc, err := NewTaxiService(newTaxiOpts)
	if err != nil {
		t.Error(err)
	}

	// found taxi with LBTC
	addr, _ := taxiSvc.getAddresses()
	_, err = faucet(addr, explorerEndpoint)
	if err != nil {
		t.Error(nil)
	}
	time.Sleep(5 * time.Second)

	topup0, err := taxiSvc.TopupWithAsset(superToken, 5000, defaultSatsPerByte)
	if err != nil {
		t.Error(err)
	}

	adminSvc := NewAdminService(taxiSvc)
	adminSvc.ChangeSpread(12)

	topup1, err := taxiSvc.TopupWithAsset(superToken, 5000, defaultSatsPerByte)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, int32(10), topup0.BasisPoint)
	assert.Equal(t, int32(12), topup1.BasisPoint)
}
