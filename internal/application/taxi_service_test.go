package application

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

const (
	superToken = "2e4c430667f861d010b6c78ce6a3943d35d84ca77ca1898031b8799c4798bc4f"
)

var (
	regtest = network.Regtest
	words   = strings.Split(mnemonic, " ")
)

func TestTopupWithAsset(t *testing.T) {
	newTaxiOpts := NewTaxiServiceArgs{
		ExplorerEndpoint: explorerEndpoint,
		Mnemonic:         words,
		Network:          regtest,
		BasisPoint:       10,
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

	topup, err := taxiSvc.TopupWithAsset(superToken, 5000, 1)
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
