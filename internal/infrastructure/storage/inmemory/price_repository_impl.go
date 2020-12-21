package inmemory

import (
	"context"
	"sync"

	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

type inmemoryPriceRepository struct {
	assetToAmountMap map[string]domain.Price
	locker           sync.Locker
}

// NewInmemoryPriceRepository creates a new inmemory PriceRepository implementation
func NewInmemoryPriceRepository() domain.PriceRepository {
	return &inmemoryPriceRepository{
		assetToAmountMap: make(map[string]domain.Price),
		locker:           &sync.Mutex{},
	}
}

func (r *inmemoryPriceRepository) UpdateOrCreatePrice(_ context.Context, price domain.Price) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	r.assetToAmountMap[price.AssetHash] = price
	return nil
}

func (r *inmemoryPriceRepository) GetPrice(_ context.Context, assetHash string) (*domain.Price, bool, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	if price, ok := r.assetToAmountMap[assetHash]; ok {
		return &price, true, nil
	}

	return nil, false, nil
}

func (r *inmemoryPriceRepository) GetAllPrices(_ context.Context) ([]domain.Price, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	prices := make([]domain.Price, 0)

	for _, value := range r.assetToAmountMap {
		prices = append(prices, value)
	}

	return prices, nil
}
