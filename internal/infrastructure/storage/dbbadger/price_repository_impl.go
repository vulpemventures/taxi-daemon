package dbbadger

import (
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
