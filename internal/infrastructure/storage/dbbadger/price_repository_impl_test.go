package dbbadger

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

var (
	price0 = domain.Price{
		AssetHash: assetHash,
		Amount:    decimal.NewFromFloat(10.54),
	}

	price1 = domain.Price{
		AssetHash: assetHash,
		Amount:    decimal.NewFromFloat(11.01),
	}

	priceBis = domain.Price{
		AssetHash: assetHashBis,
		Amount:    decimal.NewFromFloat(10.0),
	}
)

func TestUpdateOrCreatePrice(t *testing.T) {
	dirManagementForTest(t)
	repo, err := NewBadgerPriceRepository(testDataDir, nil)

	// should create
	err = repo.UpdateOrCreatePrice(context.Background(), price0)
	if err != nil {
		t.Fatal(err)
	}

	price, err := repo.GetPrice(context.Background(), price0.AssetHash)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, price.Amount.Equal(price0.Amount))

	// should update
	err = repo.UpdateOrCreatePrice(context.Background(), price1)
	if err != nil {
		t.Fatal(err)
	}

	price, err = repo.GetPrice(context.Background(), price0.AssetHash)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, price.Amount.Equal(price1.Amount))
}

func TestGetAllPrices(t *testing.T) {
	dirManagementForTest(t)
	repo, err := NewBadgerPriceRepository(testDataDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.UpdateOrCreatePrice(context.Background(), price0)
	if err != nil {
		t.Fatal(err)
	}
	err = repo.UpdateOrCreatePrice(context.Background(), priceBis)
	if err != nil {
		t.Fatal(err)
	}

	prices, err := repo.GetAllPrices(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(prices))
}

func TestGetPrice(t *testing.T) {
	dirManagementForTest(t)
	repo, err := NewBadgerPriceRepository(testDataDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.UpdateOrCreatePrice(context.Background(), price0)
	if err != nil {
		t.Fatal(err)
	}

	price, err := repo.GetPrice(context.Background(), price0.AssetHash)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, price)
}
