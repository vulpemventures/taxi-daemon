package dbbadger

import (
	"context"
	"path/filepath"

	"github.com/dgraph-io/badger/v2"
	"github.com/timshannon/badgerhold/v2"
	"github.com/vulpemventures/taxi-daemon/internal/domain"
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
	var locks []domain.Lock = []domain.Lock{}
	err := r.store.Find(&locks, nil)
	return locks, err
}

func (r *badgerLockRepository) Lock(ctx context.Context, keys []domain.Lock) error {
	for _, lock := range keys {
		err := r.store.Insert(lock.String(), lock)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *badgerLockRepository) Unlock(ctx context.Context, key domain.Lock) error {
	return r.store.Delete(key.String(), key)
}

func (r *badgerLockRepository) IsLocked(ctx context.Context, key domain.Lock) bool {
	query := badgerhold.Where("TxID").Eq(key.TxID).And("Vout").Eq(key.Vout)

	err := r.store.FindOne(&domain.Lock{}, query)
	if err != nil {
		return false
	}

	return true
}
