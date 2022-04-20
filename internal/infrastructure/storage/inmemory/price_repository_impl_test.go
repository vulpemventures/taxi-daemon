package inmemory

// import (
// 	"context"
// 	"testing"

// 	"github.com/shopspring/decimal"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
// )

// const (
// 	assetHash    = "youaretryingtoreadthisstringconstant"
// 	assetHashBis = "zizouzizouzizouzizouzizouzizouziz"
// )

// var (
// 	price0 = domain.Price{
// 		AssetHash: assetHash,
// 		Amount:    decimal.NewFromFloat(10.54),
// 	}

// 	price1 = domain.Price{
// 		AssetHash: assetHash,
// 		Amount:    decimal.NewFromFloat(11.01),
// 	}

// 	priceBis = domain.Price{
// 		AssetHash: assetHashBis,
// 		Amount:    decimal.NewFromFloat(10.0),
// 	}
// )

// func TestUpdateOrCreatePrice(t *testing.T) {
// 	repo := newInmemoryPriceRepository()

// 	// should create
// 	err := repo.UpdateOrCreatePrice(context.Background(), price0)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	price, ok := repo.assetToPriceMap[price0.AssetHash]
// 	assert.Equal(t, true, ok)
// 	assert.Equal(t, true, price.Amount.Equal(price0.Amount))

// 	// should update
// 	err = repo.UpdateOrCreatePrice(context.Background(), price1)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	price, ok = repo.assetToPriceMap[price0.AssetHash]
// 	assert.Equal(t, true, ok)
// 	assert.Equal(t, true, price.Amount.Equal(price1.Amount))
// }

// func TestGetAllPrices(t *testing.T) {
// 	repo := newInmemoryPriceRepository()
// 	err := repo.UpdateOrCreatePrice(context.Background(), price0)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	err = repo.UpdateOrCreatePrice(context.Background(), priceBis)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	prices, err := repo.GetAllPrices(context.Background())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	assert.Equal(t, 2, len(prices))
// }

// func TestGetPrice(t *testing.T) {
// 	repo := newInmemoryPriceRepository()
// 	err := repo.UpdateOrCreatePrice(context.Background(), price0)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	price, err := repo.GetPrice(context.Background(), price0.AssetHash)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	assert.NotNil(t, price)

// 	// should return nil if the price is not found in the repository
// 	price, err = repo.GetPrice(context.Background(), priceBis.AssetHash)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	assert.Nil(t, price)
// }
