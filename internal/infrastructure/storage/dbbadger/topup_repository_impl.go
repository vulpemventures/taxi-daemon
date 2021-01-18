package dbbadger

import (
	"context"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/timshannon/badgerhold/v2"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/pkg/testutils"
)

// TopupStoreKey is the directory name of the db files
const TopupStoreKey = "topup"

type badgerTopupRepository struct {
	store *badgerhold.Store
}

// NewBadgerTopupRepository is a factory function for TopupRepository (Badger type)
func NewBadgerTopupRepository(baseDirPath string, logger badger.Logger) (domain.TopupRepository, error) {
	topupStore, err := createDb(filepath.Join(baseDirPath, TopupStoreKey), logger)
	if err != nil {
		return nil, err
	}

	return &badgerTopupRepository{
		store: topupStore,
	}, nil
}

func (r *badgerTopupRepository) AddTopup(ctx context.Context, topup domain.Topup) error {
	handler := func() (interface{}, error) {
		return nil, r.store.Insert(topup.ID, topup)
	}

	_, err := runTransaction(handler)
	return err
}

func (r *badgerTopupRepository) GetAllTopups(ctx context.Context) ([]domain.Topup, error) {
	handler := func() (interface{}, error) {
		var topups []domain.Topup = []domain.Topup{}
		err := r.store.Find(&topups, nil)
		if err != nil {
			return nil, err
		}

		return topups, nil
	}

	res, err := runTransaction(handler)
	if err != nil {
		return nil, err
	}

	topups := res.([]domain.Topup)
	return topups, err
}

func (r *badgerTopupRepository) MarkAsCompleted(ctx context.Context, topupID string, partial string, assetAmount uint64) (*domain.Topup, error) {
	defer testutils.TimeTrack(time.Now(), "markAsCompleted")
	var topup domain.Topup

	err := r.store.Get(topupID, &topup)
	if err != nil {
		return nil, err
	}

	err = topup.Complete(partial, assetAmount)
	if err != nil {
		return nil, err
	}

	err = r.store.Update(topup.ID, topup)
	if err != nil {
		return nil, err
	}

	return &topup, nil
}

func (r *badgerTopupRepository) MarkAsSettled(ctx context.Context, topupID string, txID string) (*domain.Topup, error) {
	var topup domain.Topup

	err := r.store.Get(topupID, &topup)
	if err != nil {
		return nil, err
	}

	err = topup.Settle()
	if err != nil {
		return nil, err
	}

	err = r.store.Update(topup.ID, topup)
	if err != nil {
		return nil, err
	}

	return &topup, nil
}
