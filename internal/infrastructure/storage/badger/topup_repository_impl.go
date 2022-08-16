package dbbadger

import (
	"context"
	"fmt"

	badgerhold "github.com/timshannon/badgerhold/v4"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
)

type topupRepository struct {
	store *badgerhold.Store
}

func NewTopupRepository(store *badgerhold.Store) domain.TopupRepository {
	return newTopupRepository(store)
}

func newTopupRepository(store *badgerhold.Store) *topupRepository {
	return &topupRepository{store}
}

func (r *topupRepository) AddTopup(
	ctx context.Context, topup *domain.Topup,
) error {
	if err := r.store.Insert(topup.ID, topup); err != nil {
		if err == badgerhold.ErrKeyExists {
			return fmt.Errorf("topup with id %s already existing", topup.ID)
		}
		return err
	}
	return nil
}

func (r *topupRepository) GetAllTopups(
	ctx context.Context,
) ([]*domain.Topup, error) {
	topups := make([]*domain.Topup, 0)
	err := r.store.Find(&topups, nil)
	return topups, err
}

func (r *topupRepository) UpdateTopup(
	ctx context.Context, topupID string,
	updateFn func(*domain.Topup) (*domain.Topup, error),
) error {
	topup, err := r.getTopup(ctx, topupID)
	if err != nil {
		return err
	}

	updatedTopup, err := updateFn(topup)
	if err != nil {
		return err
	}

	return r.store.Update(topup.ID, *updatedTopup)
}

func (r *topupRepository) getTopup(
	_ context.Context, topupID string,
) (*domain.Topup, error) {
	var topup domain.Topup
	if err := r.store.Get(topupID, &topup); err != nil {
		return nil, err
	}
	return &topup, nil
}

func (r *topupRepository) close() {
	r.store.Close()
}
