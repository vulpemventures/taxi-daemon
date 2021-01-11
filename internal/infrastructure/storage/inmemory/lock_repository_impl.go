package inmemory

import (
	"context"
	"sync"

	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

type inmemoryLockRepository struct {
	locks  map[domain.Lock]bool
	locker sync.Locker
}

// NewInmemoryLockRepository creates a LockRepository implementation
func NewInmemoryLockRepository() domain.LockRepository {
	return newInmemoryLockRepository()
}

func newInmemoryLockRepository() *inmemoryLockRepository {
	return &inmemoryLockRepository{
		locks:  make(map[domain.Lock]bool),
		locker: &sync.Mutex{},
	}
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
