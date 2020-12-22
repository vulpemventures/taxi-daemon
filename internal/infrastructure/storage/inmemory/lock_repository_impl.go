package inmemory

import (
	"context"
	"sync"

	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

type inmemoryLockRepository struct {
	utxosKeys map[domain.Lock]bool
	locker    sync.Locker
}

// NewInmemoryLockRepository creates a LockRepository implementation
func NewInmemoryLockRepository() domain.LockRepository {
	return newInmemoryLockRepository()
}

func newInmemoryLockRepository() *inmemoryLockRepository {
	return &inmemoryLockRepository{
		utxosKeys: make(map[domain.Lock]bool),
		locker:    &sync.Mutex{},
	}
}

func (r *inmemoryLockRepository) Lock(ctx context.Context, locks []domain.Lock) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	for _, k := range locks {
		r.utxosKeys[k] = true
	}

	return nil
}

func (r *inmemoryLockRepository) IsLocked(ctx context.Context, lock domain.Lock) bool {
	r.locker.Lock()
	defer r.locker.Unlock()

	_, ok := r.utxosKeys[lock]
	return ok
}

func (r *inmemoryLockRepository) Unlock(ctx context.Context, lock domain.Lock) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	delete(r.utxosKeys, lock)
	return nil
}
