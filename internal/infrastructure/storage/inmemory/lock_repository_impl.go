package inmemory

import (
	"context"
	"sync"
	"time"

	"github.com/vulpemventures/taxi-daemon/internal/domain"
)

type inmemoryLockRepository struct {
	locks   map[domain.Lock]bool
	locker  sync.Locker
	utxoTTL time.Duration
}

// NewInmemoryLockRepository creates a LockRepository implementation
func NewInmemoryLockRepository(utxoTTL time.Duration) domain.LockRepository {
	return newInmemoryLockRepository(utxoTTL)
}

func newInmemoryLockRepository(utxoTTL time.Duration) *inmemoryLockRepository {
	return &inmemoryLockRepository{
		locks:   make(map[domain.Lock]bool),
		locker:  &sync.Mutex{},
		utxoTTL: utxoTTL,
	}
}

func (r *inmemoryLockRepository) GetConfigTTL(context.Context) time.Duration {
	return r.utxoTTL
}

func (r *inmemoryLockRepository) GetLocks(ctx context.Context) ([]domain.Lock, error) {
	locks := make([]domain.Lock, len(r.locks), len(r.locks))

	index := 0
	for lock := range r.locks {
		locks[index] = lock
		index++
	}

	return locks, nil
}

func (r *inmemoryLockRepository) Lock(ctx context.Context, locks []domain.Lock) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	for _, k := range locks {
		r.locks[k] = true
		go func(l domain.Lock) {
			time.Sleep(r.utxoTTL)
			r.Unlock(ctx, l)
		}(k)

	}

	return nil
}

func (r *inmemoryLockRepository) IsLocked(ctx context.Context, lock domain.Lock) bool {
	r.locker.Lock()
	defer r.locker.Unlock()

	_, ok := r.locks[lock]
	return ok
}

func (r *inmemoryLockRepository) Unlock(ctx context.Context, lock domain.Lock) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	delete(r.locks, lock)
	return nil
}
