package application

import (
	"bytes"
	"log"
	"sort"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
)

func TestTopupWithAsset(t *testing.T) {
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
		t.Error(nil)
	}

	// found taxi with LBTC
	addr, _ := taxiSvc.getAddresses()
	_, err = faucet(addr, explorerEndpoint)
	if err != nil {
		t.Error(nil)
	}

	time.Sleep(5 * time.Second)

	topup, err := taxiSvc.TopupWithAsset(superToken, 5000, defaultSatsPerByte)
	assert.Nil(t, err)
	assert.NotNil(t, topup)

	p, err := pset.NewPsetFromBase64(topup.Partial)
	assert.Nil(t, err)
	assert.NotNil(t, p)

	assert.Equal(t, true, len(p.UnsignedTx.Outputs) >= 2)

	var payoutOutput *transaction.TxOutput

	for _, out := range p.UnsignedTx.Outputs {
		if bytes.Equal(assetHashBytes(superToken), out.Asset) {
			payoutOutput = out
		}
	}

	hex, err := p.UnsignedTx.ToHex()
	if err != nil {
		t.Error(err)
	}

	log.Print(hex)

	assert.NotNil(t, payoutOutput)
}

func TestListAssets(t *testing.T) {
	doesnotmatterAmount := decimal.NewFromFloat(66.6)

	type testFixture struct {
		initialPrices []domain.Price
		expected      []string
	}

	tests := []testFixture{
		{
			initialPrices: []domain.Price{
				domain.Price{
					AssetHash: "bitcoin",
					Amount:    doesnotmatterAmount,
				},
				domain.Price{
					AssetHash: "bobtoken",
					Amount:    doesnotmatterAmount,
				},
				domain.Price{
					AssetHash: "liquidbitcoin",
					Amount:    doesnotmatterAmount,
				},
				domain.Price{
					AssetHash: "solidbitcoin",
					Amount:    doesnotmatterAmount,
				},
			},
			expected: []string{"bitcoin", "bobtoken", "liquidbitcoin", "solidbitcoin"},
		},
	}

	for _, test := range tests {
		newTaxiOpts := NewTaxiServiceArgs{
			ExplorerEndpoint: explorerEndpoint,
			Mnemonic:         words,
			Network:          regtest,
			BasisPoint:       10,
			TopupRepository:  inmemory.NewInmemoryTopupRepository(),
			InitialPrices:    test.initialPrices,
		}

		taxiSvc, err := NewTaxiService(newTaxiOpts)
		if err != nil {
			t.Error(err)
		}

		assets := taxiSvc.ListAssets()
		sort.Strings(assets)
		sort.Strings(test.expected)
		assert.Equal(t, test.expected, assets)
	}

}
