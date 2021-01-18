package dbbadger

import (
	"context"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/timshannon/badgerhold/v2"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

// LockStoreKey is the directory name of the db files
const LockStoreKey = "lock"

type badgerLockRepository struct {
	store   *badgerhold.Store
	utxoTTL time.Duration
}

// NewBadgerLockRepository is a factory function for Badger LockRepository implementation
func NewBadgerLockRepository(baseDirPath string, utxoTTL time.Duration, logger badger.Logger) (domain.LockRepository, error) {
	lockStore, err := createDb(filepath.Join(baseDirPath, LockStoreKey), logger)
	if err != nil {
		return nil, err
	}

	return &badgerLockRepository{
		store:   lockStore,
		utxoTTL: utxoTTL,
	}, nil
}

func (r *badgerLockRepository) GetConfigTTL(context.Context) time.Duration {
	return r.utxoTTL
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
	encKey, err := encodeKey(key.String(), "Lock")
	if err != nil {
		return err
	}

	encData, err := badgerhold.DefaultEncode(key)
	if err != nil {
		return err
	}

	badgerEntryWithTTL := badger.NewEntry(encKey, encData).WithTTL(r.utxoTTL)
	handler := func() (interface{}, error) {
		return nil, r.store.Badger().Update(func(tx *badger.Txn) error {
			return tx.SetEntry(badgerEntryWithTTL)
		})
	}

	_, err = runTransaction(handler)
	return err
}

func (r *badgerLockRepository) Unlock(ctx context.Context, key domain.Lock) error {
	handler := func() (interface{}, error) {
		err := r.store.Delete(key.String(), key)
		return nil, err
	}

	_, err := runTransaction(handler)
	return err
}

func (r *badgerLockRepository) IsLocked(ctx context.Context, key domain.Lock) bool {
	handler := func() (interface{}, error) {
		var lock domain.Lock
		err := r.store.Get(key.String(), &lock)
		return lock, err
	}

	_, err := runTransaction(handler)
	return err == nil
}
