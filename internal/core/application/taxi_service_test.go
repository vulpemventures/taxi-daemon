package application_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vulpemventures/taxi-daemon/internal/core/application"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
	dbbadger "github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/badger"
)

var (
	ctx             = context.Background()
	estimatedSize   = uint64(400)
	millisatPerByte = uint64(100)
	knownPayout     = "el1qq2al6wn0rxt3t64mhw5m7eznxsxj8j6y3h59c96sswc3fxjt2kvc6axwelge78wpxgaa0sc626zsqlm23etdg2876ptxjxp8r"
)

func TestTaxiService(t *testing.T) {
	mockedWallet := &mockWallet{}
	mockedWallet.On(
		"CreatePartialWithFeesAndPayout",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(
		randomPartial(), nil, uint64(time.Now().Add(5*time.Minute).Unix()),
		uint64(400), uint64(15150000), nil,
	)
	mockedWallet.On(
		"UpdatePartialWithFees", mock.Anything, mock.Anything, mock.Anything,
	).Return(randomPartial(), uint64(time.Now().Add(5*time.Minute).Unix()), nil)
	mockedWallet.On("PartialContainsKnownPayout", mock.Anything, mock.Anything).
		Return(true, nil)
	mockedWallet.On("PartialContainsKnownUtxos", mock.Anything, mock.Anything).
		Return(true, nil)

	repoManager, err := newRepoManagerForTaxiService()
	require.NoError(t, err)

	taxiSvc := application.NewTaxiService(mockedWallet, repoManager)

	prices, err := taxiSvc.ListAssets(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, prices)

	asset := prices[0].AssetHash
	topupWithAsset, _, err := taxiSvc.TopupWithAsset(
		ctx, asset, estimatedSize, millisatPerByte,
	)
	require.NoError(t, err)
	require.NotNil(t, topupWithAsset)

	partialWithKnownPayout := randomPartial()
	topupWithKnownPayout, err := taxiSvc.TopupWithKnownPayout(
		ctx, partialWithKnownPayout, millisatPerByte,
	)
	require.NoError(t, err)
	require.NotNil(t, topupWithKnownPayout)
}

func newRepoManagerForTaxiService() (ports.RepoManager, error) {
	rm, err := dbbadger.NewRepoManager("", nil)
	if err != nil {
		return nil, err
	}

	if err := rm.PriceRepository().UpsertPrice(ctx, &domain.Price{
		AssetHash:  randomAsset(),
		Amount:     decimal.NewFromInt(45000),
		BasisPoint: 100,
	}); err != nil {
		return nil, err
	}

	if _, err := rm.WhitelistRepository().AddAddresses(
		ctx, []string{knownPayout},
	); err != nil {
		return nil, err
	}

	return rm, nil
}

func randomPartial() string {
	return base64.StdEncoding.EncodeToString(randomBytes(100))
}

func randomAsset() string {
	return hex.EncodeToString(randomBytes(32))
}

func randomBytes(len int) []byte {
	b := make([]byte, len)
	rand.Read(b)
	return b
}
