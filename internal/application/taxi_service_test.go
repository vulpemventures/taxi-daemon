package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
)

func TestTopupWithAsset(t *testing.T) {
	newTaxiOpts := NewTaxiServiceArgs{
		Mnemonic:        words,
		Network:         regtest,
		FixedFeeAmount:  100000000,
		TopupRepository: inmemory.NewInmemoryTopupRepository(),
		LockRepository:  inmemory.NewInmemoryLockRepository(),
		PriceRepository: newMockedPriceRepository(t, []string{superToken}),
		ExplorerSvc:     explorer.NewService(explorerEndpoint),
	}

	taxiSvc, err := newTaxiService(newTaxiOpts)
	if err != nil {
		t.Fatal(nil)
	}

	// found taxi with LBTC
	addr, err := taxiSvc.getFundAddress()
	if err != nil {
		t.Fatal(err)
	}
	_, err = faucet(addr, explorerEndpoint)
	if err != nil {
		t.Fatal(nil)
	}

	time.Sleep(10 * time.Second)

	topup, err := taxiSvc.TopupWithAsset(context.Background(), superToken)
	assert.Nil(t, err)
	assert.NotNil(t, topup)

	p, err := pset.NewPsetFromBase64(topup.Partial)
	assert.Nil(t, err)
	assert.NotNil(t, p)

	assert.Equal(t, true, len(p.UnsignedTx.Outputs) >= 2)

	var payoutOutput *transaction.TxOutput
	var feeOutput *transaction.TxOutput

	for _, out := range p.UnsignedTx.Outputs {
		if bytes.Equal(assetHashBytes(superToken), out.Asset) {
			payoutOutput = out
		}

		if bytes.Equal(out.Script, []byte{}) {
			feeOutput = out
		}
	}

	assert.NotNil(t, payoutOutput)
	assert.NotNil(t, feeOutput)
	assert.Equal(t, 1, len(p.UnsignedTx.Inputs))
}

func TestListAssets(t *testing.T) {
	type testFixture struct {
		input    []string
		expected []string
	}

	tests := []testFixture{
		{
			input:    []string{"bitcoin", "bobtoken", "liquidbitcoin", "solidbitcoin"},
			expected: []string{"bitcoin", "bobtoken", "liquidbitcoin", "solidbitcoin"},
		},
	}

	for _, test := range tests {
		newTaxiOpts := NewTaxiServiceArgs{
			Mnemonic:        words,
			Network:         regtest,
			FixedFeeAmount:  100000000,
			TopupRepository: inmemory.NewInmemoryTopupRepository(),
			LockRepository:  inmemory.NewInmemoryLockRepository(),
			PriceRepository: newMockedPriceRepository(t, test.input),
			ExplorerSvc:     explorer.NewService(explorerEndpoint),
		}

		taxiSvc, err := NewTaxiService(newTaxiOpts)
		if err != nil {
			t.Fatal(err)
		}

		assets := taxiSvc.ListAssets(context.Background())
		sort.Strings(assets)
		sort.Strings(test.expected)
		assert.Equal(t, test.expected, assets)
	}

}

func assetHashBytes(asset string) []byte {
	buf, _ := hex.DecodeString(asset)
	assetBuf := append([]byte{0x01}, elementsutil.ReverseBytes(buf)...)
	return assetBuf
}
