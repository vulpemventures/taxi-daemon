package inmemory

import (
	"context"
	"errors"
	"sync"

	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

type inmemoryTopupRepository struct {
	topups           map[string]domain.Topup
	repositoryLocker sync.Locker
}

// NewInmemoryTopupRepository creates a new inmemory TopupRepository implementation
func NewInmemoryTopupRepository() domain.TopupRepository {
	return &inmemoryTopupRepository{
		topups:           make(map[string]domain.Topup),
		repositoryLocker: &sync.Mutex{},
	}
}

func (r *inmemoryTopupRepository) AddTopup(ctx context.Context, topup domain.Topup) error {
	r.repositoryLocker.Lock()
	defer r.repositoryLocker.Unlock()

	r.topups[topup.ID] = topup

	return nil
}

func (r *inmemoryTopupRepository) GetAllTopups(ctx context.Context) ([]domain.Topup, error) {
	r.repositoryLocker.Lock()
	defer r.repositoryLocker.Unlock()

	topups := make([]domain.Topup, 0)

	for _, v := range r.topups {
		topups = append(topups, v)
	}

	return topups, nil
}

func (r *inmemoryTopupRepository) SetCompleted(ctx context.Context, topupID string, partial string, assetAmount uint64) (*domain.Topup, error) {
	r.repositoryLocker.Lock()
	defer r.repositoryLocker.Unlock()
	topup, ok := r.topups[topupID]
	if !ok {
		return nil, errors.New("Topup does not exist")
	}

	topup.Complete(partial, assetAmount)
	r.topups[topupID] = topup

	return &topup, nil
}

func (r *inmemoryTopupRepository) SetSettled(ctx context.Context, topupID string, txid string) (*domain.Topup, error) {
	r.repositoryLocker.Lock()
	defer r.repositoryLocker.Unlock()
	topup, ok := r.topups[topupID]
	if !ok {
		return nil, errors.New("Topup does not exist")
	}

	topup.Settle()
	r.topups[topupID] = topup

	return &topup, nil
}
