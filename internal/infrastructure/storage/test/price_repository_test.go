package db_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	dbbadger "github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/badger"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/inmemory"
)

var (
	price *domain.Price
	ctx   = context.Background()
)

func TestPriceRepository(t *testing.T) {
	repositories, err := newPriceRepositories()
	require.NoError(t, err)

	for name, repo := range repositories {
		t.Run(name, func(t *testing.T) {
			testPriceRepository(t, repo)
		})
	}
}

func testPriceRepository(t *testing.T, repo domain.PriceRepository) {
	testUpsertPrice(t, repo)

	testGetPrice(t, repo)
}

func testUpsertPrice(t *testing.T, repo domain.PriceRepository) {
	t.Run("upsert_price", func(t *testing.T) {
		prices, err := repo.GetAllPrices(ctx)
		require.NoError(t, err)
		require.Empty(t, prices)

		price = randomPrice()
		err = repo.UpsertPrice(ctx, price)
		require.NoError(t, err)

		prices, err = repo.GetAllPrices(ctx)
		require.NoError(t, err)
		require.Len(t, prices, 1)
		require.Equal(t, price.AssetHash, prices[0].AssetHash)
		require.Equal(t, price.Amount, prices[0].Amount)
		require.Equal(t, price.BasisPoint, prices[0].BasisPoint)

		updatedAmount := randomAmount()
		price.Amount = updatedAmount
		err = repo.UpsertPrice(ctx, price)
		require.NoError(t, err)

		prices, err = repo.GetAllPrices(ctx)
		require.NoError(t, err)
		require.Len(t, prices, 1)
		require.Equal(t, price.AssetHash, prices[0].AssetHash)
		require.Equal(t, updatedAmount, prices[0].Amount)
		require.Equal(t, price.BasisPoint, prices[0].BasisPoint)

		anotherPrice := randomPrice()
		err = repo.UpsertPrice(ctx, anotherPrice)
		require.NoError(t, err)

		prices, err = repo.GetAllPrices(ctx)
		require.NoError(t, err)
		require.Len(t, prices, 2)
	})
}

func testGetPrice(t *testing.T, repo domain.PriceRepository) {
	t.Run("get_price", func(t *testing.T) {
		asset := randomAsset()
		foundPrice, err := repo.GetPrice(ctx, asset)
		require.Error(t, err)
		require.Nil(t, foundPrice)

		foundPrice, err = repo.GetPrice(ctx, price.AssetHash)
		require.NoError(t, err)
		require.NotNil(t, foundPrice)
		require.Exactly(t, *price, *foundPrice)
	})
}

func newPriceRepositories() (map[string]domain.PriceRepository, error) {
	inmemoryRepoManager := inmemory.NewRepoManager()
	badgerRepoManager, err := dbbadger.NewRepoManager("", nil)
	if err != nil {
		return nil, err
	}

	return map[string]domain.PriceRepository{
		"inmemory": inmemoryRepoManager.PriceRepository(),
		"badger":   badgerRepoManager.PriceRepository(),
	}, nil
}

func randomPrice() *domain.Price {
	return &domain.Price{
		AssetHash:  randomAsset(),
		Amount:     randomAmount(),
		BasisPoint: randomBasisPoint(),
	}
}

func randomAsset() string {
	return hex.EncodeToString(randomBytes(32))
}

func randomAmount() decimal.Decimal {
	return decimal.NewFromInt(int64(randomIntInRange(1, 10000)))
}

func randomBasisPoint() uint32 {
	return uint32(randomIntInRange(1, 9999))
}

func randomBytes(len int) []byte {
	b := make([]byte, len)
	rand.Read(b)
	return b
}

func randomIntInRange(min, max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(int(n.Int64())) + min
}
