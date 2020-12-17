package inmemory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

type inmemoryTopupRepository struct {
	topups           map[string]domain.Topup
	repositoryLocker sync.Locker
}

func NewInmemoryTopupRepository() domain.TopupRepository {
	return &inmemoryTopupRepository{
		topups:           make(map[string]domain.Topup),
		repositoryLocker: &sync.Mutex{},
	}
}

func (r *inmemoryTopupRepository) AddTopup(ctx context.Context, topup domain.Topup) error {
	r.repositoryLocker.Lock()
	defer r.repositoryLocker.Unlock()

	r.topups[topup.Id] = topup

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

func (r *inmemoryTopupRepository) SetRequested(ctx context.Context, topupId string) (*domain.Topup, error) {
	r.repositoryLocker.Lock()
	defer r.repositoryLocker.Unlock()
	topup, ok := r.topups[topupId]
	if !ok {
		return nil, errors.New("Topup does not exist")
	}

	topup.RequestedAt = time.Now()
	topup.Status = domain.Requested

	r.topups[topupId] = topup

	return &topup, nil
}

func (r *inmemoryTopupRepository) SetCompleted(ctx context.Context, topupId string, partial string, assetAmount uint64) (*domain.Topup, error) {
	r.repositoryLocker.Lock()
	defer r.repositoryLocker.Unlock()
	topup, ok := r.topups[topupId]
	if !ok {
		return nil, errors.New("Topup does not exist")
	}

	topup.CompletedAt = time.Now()
	topup.Status = domain.Completed
	topup.Partial = partial
	topup.AssetAmount = assetAmount

	r.topups[topupId] = topup

	return &topup, nil
}

func (r *inmemoryTopupRepository) SetSettled(ctx context.Context, topupId string, txid string) (*domain.Topup, error) {
	r.repositoryLocker.Lock()
	defer r.repositoryLocker.Unlock()
	topup, ok := r.topups[topupId]
	if !ok {
		return nil, errors.New("Topup does not exist")
	}

	timestamp := time.Now()
	topup.SettledAt = timestamp
	topup.Status = domain.Settled

	r.topups[topupId] = topup

	return &topup, nil
}
