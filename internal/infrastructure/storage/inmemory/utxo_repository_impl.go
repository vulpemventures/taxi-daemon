package inmemory

import (
	"context"
	"sync"

	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

type inmemoryUtxoRepository struct {
	utxosKeys map[domain.UtxoKey]bool
	locker    sync.Locker
}

func NewInmemoryUtxoKeyRepository() domain.UtxoKeyRepository {
	return &inmemoryUtxoRepository{
		utxosKeys: make(map[domain.UtxoKey]bool),
		locker:    &sync.Mutex{},
	}
}

func (r *inmemoryUtxoRepository) AddKeys(ctx context.Context, keys []domain.UtxoKey) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	for _, k := range keys {
		r.utxosKeys[k] = true
	}

	return nil
}

func (r *inmemoryUtxoRepository) Contains(ctx context.Context, key domain.UtxoKey) bool {
	r.locker.Lock()
	defer r.locker.Unlock()

	_, ok := r.utxosKeys[key]
	return ok
}

func (r *inmemoryUtxoRepository) Remove(ctx context.Context, key domain.UtxoKey) error {
	r.locker.Lock()
	defer r.locker.Unlock()

	delete(r.utxosKeys, key)
	return nil
}
