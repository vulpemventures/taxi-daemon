package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
)

type topupRepository struct {
	topups map[string]*domain.Topup
	lock   *sync.RWMutex
}

// NewTopupRepository creates a new inmemory TopupRepository implementation
func NewTopupRepository() domain.TopupRepository {
	return newTopupRepository()
}

func newTopupRepository() *topupRepository {
	return &topupRepository{
		topups: make(map[string]*domain.Topup),
		lock:   &sync.RWMutex{},
	}
}

func (r *topupRepository) AddTopup(_ context.Context, topup *domain.Topup) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, ok := r.topups[topup.ID]; ok {
		return fmt.Errorf("topup with id %s already existing", topup.ID)
	}
	r.topups[topup.ID] = topup
	return nil
}

func (r *topupRepository) GetAllTopups(ctx context.Context) ([]*domain.Topup, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	topups := make([]*domain.Topup, 0)
	for _, v := range r.topups {
		topups = append(topups, v)
	}
	return topups, nil
}

func (r *topupRepository) UpdateTopup(
	ctx context.Context, topupID string,
	updatefn func(*domain.Topup) (*domain.Topup, error),
) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	topup, ok := r.topups[topupID]
	if !ok {
		return fmt.Errorf("topup %s does not exist", topupID)
	}

	updatedTopup, err := updatefn(topup)
	if err != nil {
		return err
	}

	r.topups[topupID] = updatedTopup
	return nil
}
