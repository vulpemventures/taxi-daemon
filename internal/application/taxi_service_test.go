package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
)

func TestUtxoTTLAfterTopup(t *testing.T) {
	lockRepo := inmemory.NewInmemoryLockRepository()
	newWords, err := generateMnemonic(256)
	if err != nil {
		t.Fatal(err)
	}

	newTaxiOpts := NewTaxiServiceArgs{
		Mnemonic:        newWords,
		Network:         regtest,
		FixedFeeAmount:  100000000,
		TopupRepository: inmemory.NewInmemoryTopupRepository(),
		LockRepository:  lockRepo,
		PriceRepository: newMockedPriceRepository(t, []string{superToken}),
		ExplorerSvc:     explorer.NewService(explorerEndpoint),
		UtxoTTL:         3 * time.Second,
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

	topup, err := taxiSvc.TopupWithAsset(context.Background(), superToken)
	assert.Nil(t, err)
	assert.NotNil(t, topup)

	p, err := pset.NewPsetFromBase64(topup.Partial)
	assert.Nil(t, err)
	assert.NotNil(t, p)

	topup, err = taxiSvc.TopupWithAsset(context.Background(), superToken)
	if err == nil {
		t.Fatal(err)
	}

	time.Sleep(4 * time.Second)

	topup, err = taxiSvc.TopupWithAsset(context.Background(), superToken)
	assert.Nil(t, err)
	assert.NotNil(t, topup)
}

func TestTopupWithAsset(t *testing.T) {
	hdTreeRepo := newMockedHDTreeRepository(0)
	lockRepo := inmemory.NewInmemoryLockRepository()
	topupRepo := newMockedTopupRepository(nil)
	priceRepo := newMockedPriceRepository([]string{superToken})
	explorerSvc := explorer.NewService(explorerEndpoint)

	config.Set(config.MnemonicKey, mnemonic)
	t.Cleanup(func() {
		config.Set(config.MnemonicKey, "")
	})

	taxiSvc := newTaxiService(
		hdTreeRepo,
		lockRepo,
		topupRepo,
		priceRepo,
		explorerSvc,
	)
	ctx := context.Background()

	addresses, err := hdTreeRepo.GetNewFundingAddresses(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	// fund taxi with LBTC
	if _, err := faucet(addresses[0], explorerEndpoint); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	topup, err := taxiSvc.TopupWithAsset(ctx, superToken)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, topup)

	p, err := pset.NewPsetFromBase64(topup.Partial)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(p.UnsignedTx.Inputs))
	assert.Equal(t, 2, len(p.UnsignedTx.Outputs))

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
}

func TestListAssets(t *testing.T) {
	hdTreeRepo := newMockedHDTreeRepository(0)
	lockRepo := inmemory.NewInmemoryLockRepository()
	topupRepo := newMockedTopupRepository(nil)
	priceRepo := newMockedPriceRepository([]string{superToken, hyperToken})
	explorerSvc := explorer.NewService(explorerEndpoint)

	taxiSvc := newTaxiService(
		hdTreeRepo,
		lockRepo,
		topupRepo,
		priceRepo,
		explorerSvc,
	)
	ctx := context.Background()

	assets, err := taxiSvc.ListAssets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(assets))

}

func assetHashBytes(asset string) []byte {
	buf, _ := hex.DecodeString(asset)
	assetBuf := append([]byte{0x01}, elementsutil.ReverseBytes(buf)...)
	return assetBuf
}
