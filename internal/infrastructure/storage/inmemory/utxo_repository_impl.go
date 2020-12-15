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

func NewInmemoryLockRepository() domain.LockRepository {
	return &inmemoryLockRepository{
		utxosKeys: make(map[domain.Lock]bool),
		locker:    &sync.Mutex{},
	}
}

func (r *inmemoryLockRepository) Add(ctx context.Context, keys []domain.Lock) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	for _, k := range keys {
		r.utxosKeys[k] = true
	}

	return nil
}

func (r *inmemoryLockRepository) Contains(ctx context.Context, key domain.Lock) bool {
	r.locker.Lock()
	defer r.locker.Unlock()

	_, ok := r.utxosKeys[key]
	return ok
}

func (r *inmemoryLockRepository) Remove(ctx context.Context, key domain.Lock) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	delete(r.utxosKeys, key)
	return nil
}
