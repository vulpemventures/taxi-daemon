package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
)

type priceRepository struct {
	prices map[string]*domain.Price
	lock   *sync.RWMutex
}

// NewPriceRepository creates a new inmemory PriceRepository implementation
func NewPriceRepository() domain.PriceRepository {
	return newPriceRepository()
}

func newPriceRepository() *priceRepository {
	return &priceRepository{
		prices: make(map[string]*domain.Price),
		lock:   &sync.RWMutex{},
	}
}

func (r *priceRepository) UpsertPrice(_ context.Context, price *domain.Price) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.prices[price.AssetHash] = price
	return nil
}

func (r *priceRepository) GetPrice(_ context.Context, assetHash string) (*domain.Price, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	price, ok := r.prices[assetHash]
	if !ok {
		return nil, fmt.Errorf("price not found for asset hash %s", assetHash)
	}
	return price, nil
}

func (r *priceRepository) GetAllPrices(_ context.Context) ([]*domain.Price, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	prices := make([]*domain.Price, 0, len(r.prices))
	for _, p := range r.prices {
		prices = append(prices, p)
	}
	return prices, nil
}
