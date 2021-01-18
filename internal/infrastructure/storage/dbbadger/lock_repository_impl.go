package dbbadger

import (
	"context"
	"path/filepath"

	"github.com/dgraph-io/badger/v2"
	"github.com/timshannon/badgerhold/v2"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

// LockStoreKey is the directory name of the db files
const LockStoreKey = "lock"

type badgerLockRepository struct {
	store *badgerhold.Store
}

// NewBadgerLockRepository is a factory function for Badger LockRepository implementation
func NewBadgerLockRepository(baseDirPath string, logger badger.Logger) (domain.LockRepository, error) {
	lockStore, err := createDb(filepath.Join(baseDirPath, LockStoreKey), logger)
	if err != nil {
		return nil, err
	}

	return &badgerLockRepository{
		store: lockStore,
	}, nil
}

func (r *badgerLockRepository) GetLocks(ctx context.Context) ([]domain.Lock, error) {
	handler := func() (interface{}, error) {
		var locks []domain.Lock = []domain.Lock{}
		err := r.store.Find(&locks, nil)
		return locks, err
	}

	result, err := runTransaction(handler)

	return result.([]domain.Lock), err
}

func (r *badgerLockRepository) Lock(ctx context.Context, keys []domain.Lock) error {
	for _, k := range keys {
		err := r.lock(k)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *badgerLockRepository) lock(key domain.Lock) error {
	handler := func() (interface{}, error) {
		return nil, r.store.Insert(key.String(), key)
	}

	_, err := runTransaction(handler)
	return err
}

func (r *badgerLockRepository) Unlock(ctx context.Context, key domain.Lock) error {
	txHandler := func() (interface{}, error) {
		err := r.store.Delete(key.String(), key)
		return nil, err
	}

	_, err := runTransaction(txHandler)
	return err
}

func (r *badgerLockRepository) IsLocked(ctx context.Context, key domain.Lock) bool {
	query := badgerhold.Where("TxID").Eq(key.TxID).And("Vout").Eq(key.Vout)

	handler := func() (interface{}, error) {
		err := r.store.FindOne(&domain.Lock{}, query)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err := runTransaction(handler)
	return err == nil
}
