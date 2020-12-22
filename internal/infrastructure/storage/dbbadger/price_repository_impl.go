package dbbadger

import (
	"context"
	"path/filepath"

	"github.com/dgraph-io/badger/v2"
	"github.com/timshannon/badgerhold/v2"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

// PriceStoreKey is the directory name of the db files
const PriceStoreKey = "price"

type badgerPriceRepository struct {
	store *badgerhold.Store
}

// NewBadgerPriceRepository is a factory function for Badger LockRepository implementation
func NewBadgerPriceRepository(baseDirPath string, logger badger.Logger) (domain.PriceRepository, error) {
	lockStore, err := createDb(filepath.Join(baseDirPath, PriceStoreKey), logger)
	if err != nil {
		return nil, err
	}

	return &badgerPriceRepository{
		store: lockStore,
	}, nil
}

func (r *badgerPriceRepository) UpdateOrCreatePrice(ctx context.Context, price domain.Price) error {

}

func (r *badgerPriceRepository) GetPrice(ctx context.Context, assetHash string) (*domain.Price, error) {

}

func (r *badgerPriceRepository) GetAllPrices(ctx context.Context) ([]domain.Price, error) {

}
