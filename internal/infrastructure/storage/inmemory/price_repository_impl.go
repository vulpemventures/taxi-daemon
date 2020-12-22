package inmemory

import (
	"context"
	"sync"

	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

type inmemoryPriceRepository struct {
	assetToPriceMap map[string]domain.Price
	locker          sync.Locker
}

// NewInmemoryPriceRepository creates a new inmemory PriceRepository implementation
func NewInmemoryPriceRepository() domain.PriceRepository {
	return newInmemoryPriceRepository()
}

func newInmemoryPriceRepository() *inmemoryPriceRepository {
	return &inmemoryPriceRepository{
		assetToPriceMap: make(map[string]domain.Price),
		locker:          &sync.Mutex{},
	}
}

func (r *inmemoryPriceRepository) UpdateOrCreatePrice(_ context.Context, price domain.Price) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	r.assetToPriceMap[price.AssetHash] = price
	return nil
}

func (r *inmemoryPriceRepository) GetPrice(_ context.Context, assetHash string) (*domain.Price, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	if price, ok := r.assetToPriceMap[assetHash]; ok {
		return &price, nil
	}

	return nil, nil
}

func (r *inmemoryPriceRepository) GetAllPrices(_ context.Context) ([]domain.Price, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	prices := make([]domain.Price, len(r.assetToPriceMap), len(r.assetToPriceMap))

        i := 0
	for _, p := range r.assetToPriceMap {
		prices[i] = p
		i++
	}

	return prices, nil
}
