package inmemory

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

type inmemoryPriceRepository struct {
	assetToAmountMap map[string]decimal.Decimal
	locker           sync.Locker
}

func NewInmemoryPriceRepository() domain.PriceRepository {
	return &inmemoryPriceRepository{
		assetToAmountMap: make(map[string]decimal.Decimal),
		locker:           &sync.Mutex{},
	}
}

func (r *inmemoryPriceRepository) UpdateOrCreatePrice(_ context.Context, price domain.Price) error {
	r.locker.Lock()
	defer r.locker.Unlock()
	r.assetToAmountMap[price.AssetHash] = price.Amount
	return nil
}

func (r *inmemoryPriceRepository) GetPrice(_ context.Context, assetHash string) (*domain.Price, bool, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	if amount, ok := r.assetToAmountMap[assetHash]; ok {
		return &domain.Price{
			AssetHash: assetHash,
			Amount:    amount,
		}, true, nil
	}

	return nil, false, nil
}
