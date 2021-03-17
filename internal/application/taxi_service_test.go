package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	explorer "github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/taxi-daemon/internal/domain"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/inmemory"
	"github.com/vulpemventures/taxi-daemon/pkg/testutils"
)

func TestTopupWithAsset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	hdTreeRepo := newMockedHDTreeRepository(0)
	lockRepo := inmemory.NewInmemoryLockRepository(utxoTTL)
	topupRepo := newMockedTopupRepository(nil)
	priceRepo := newMockedPriceRepository([]string{superToken})
	explorerSvc, err := explorer.NewService(explorerEndpoint)
	if err != nil {
		t.Fatal(err)
	}

	cache := newCache(NewCacheArgs{
		Explorer:         explorerSvc,
		HdTreeRepository: hdTreeRepo,
		LockRepository:   lockRepo,
		FragmentAsset:    regtest.AssetID,
		FragmentAmount:   fixedFeeAmount,
		Interval:         30 * time.Second,
	})

	args := NewTaxiServiceArgs{
		hdTreeRepo,
		lockRepo,
		topupRepo,
		priceRepo,
		regtest,
		fixedFeeAmount,
		words,
		cache,
	}

	taxiSvc := newTaxiService(args)
	ctx := context.Background()

	addresses, err := hdTreeRepo.GetNewFundingAddresses(ctx, 3, false)
	if err != nil {
		t.Fatal(err)
	}

	// fund taxi with LBTC
	if err := testutils.Fragment(addresses); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Second)

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

	inLock := domain.Lock{
		TxID: hex.EncodeToString(elementsutil.ReverseBytes(p.UnsignedTx.Inputs[0].Hash)),
		Vout: p.UnsignedTx.Inputs[0].Index,
	}
	assert.Equal(t, true, lockRepo.IsLocked(ctx, inLock))
}

func TestListAssets(t *testing.T) {
	hdTreeRepo := newMockedHDTreeRepository(0)
	lockRepo := inmemory.NewInmemoryLockRepository(utxoTTL)
	topupRepo := newMockedTopupRepository(nil)
	priceRepo := newMockedPriceRepository([]string{superToken, hyperToken})
	explorerSvc, err := explorer.NewService(explorerEndpoint)
	if err != nil {
		t.Fatal(err)
	}

	cache := newCache(NewCacheArgs{
		Explorer:         explorerSvc,
		HdTreeRepository: hdTreeRepo,
		LockRepository:   lockRepo,
		FragmentAsset:    regtest.AssetID,
		FragmentAmount:   fixedFeeAmount,
		Interval:         30 * time.Second,
	})

	args := NewTaxiServiceArgs{
		hdTreeRepo,
		lockRepo,
		topupRepo,
		priceRepo,
		regtest,
		fixedFeeAmount,
		words,
		cache,
	}

	taxiSvc := newTaxiService(args)
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
