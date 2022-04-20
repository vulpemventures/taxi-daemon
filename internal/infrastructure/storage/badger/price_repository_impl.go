package dbbadger

import (
	"context"
	"fmt"

	badgerhold "github.com/timshannon/badgerhold/v4"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
)

type priceRepository struct {
	store *badgerhold.Store
}

func NewPriceRepository(store *badgerhold.Store) domain.PriceRepository {
	return newPriceRepository(store)
}

func newPriceRepository(store *badgerhold.Store) *priceRepository {
	return &priceRepository{store}
}

func (r *priceRepository) UpsertPrice(_ context.Context, price *domain.Price) error {
	return r.store.Upsert(price.AssetHash, *price)
}

func (r *priceRepository) GetPrice(_ context.Context, assetHash string) (*domain.Price, error) {
	var price domain.Price
	if err := r.store.Get(assetHash, &price); err != nil {
		if err == badgerhold.ErrNotFound {
			return nil, fmt.Errorf("price not found for asset hash %s", assetHash)
		}
		return nil, err
	}
	return &price, nil
}

func (r *priceRepository) GetAllPrices(_ context.Context) ([]*domain.Price, error) {
	prices := make([]domain.Price, 0)
	if err := r.store.Find(&prices, nil); err != nil {
		return nil, err
	}

	list := make([]*domain.Price, 0, len(prices))
	for _, p := range prices {
		list = append(list, &p)
	}
	return list, nil
}

func (r *priceRepository) close() {
	r.store.Close()
}
