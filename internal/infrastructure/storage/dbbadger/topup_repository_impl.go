package dbbadger

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/timshannon/badgerhold/v2"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
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
	return r.store.Insert(topup.ID, topup)
}

func (r *badgerTopupRepository) GetAllTopups(ctx context.Context) ([]domain.Topup, error) {
	var topups []domain.Topup = []domain.Topup{}
	err := r.store.Find(&topups, nil)
	return topups, err
}

func (r *badgerTopupRepository) SetRequested(ctx context.Context, topupID string) (*domain.Topup, error) {
	var topupUpdated *domain.Topup

	query := badgerhold.Where("ID").Eq(topupID)
	err := r.store.UpdateMatching(&domain.Topup{}, query, func(record interface{}) error {
		update, ok := record.(*domain.Topup)
		if !ok {
			return errors.New("record isn't the correct type! topup type is expected")
		}

		update.RequestedAt = time.Now()
		update.Status = domain.Requested

		topupUpdated = update
		return nil
	})

	return topupUpdated, err
}

func (r *badgerTopupRepository) SetCompleted(ctx context.Context, topupID string, partial string, assetAmount uint64) (*domain.Topup, error) {
	var topupUpdated *domain.Topup

	query := badgerhold.Where("ID").Eq(topupID)
	err := r.store.UpdateMatching(&domain.Topup{}, query, func(record interface{}) error {
		update, ok := record.(*domain.Topup)
		if !ok {
			return errors.New("record isn't the correct type! Need domain.Topup type")
		}

		update.CompletedAt = time.Now()
		update.Status = domain.Completed
		update.Partial = partial
		update.AssetAmount = assetAmount

		topupUpdated = update
		return nil
	})

	return topupUpdated, err
}

func (r *badgerTopupRepository) SetSettled(ctx context.Context, topupID string, txID string) (*domain.Topup, error) {
	var topupUpdated *domain.Topup

	query := badgerhold.Where("ID").Eq(topupID)
	err := r.store.UpdateMatching(&domain.Topup{}, query, func(record interface{}) error {
		update, ok := record.(*domain.Topup)
		if !ok {
			return errors.New("record isn't the correct type! Need domain.Topup type")
		}

		update.SettledAt = time.Now()
		update.Status = domain.Settled
		// TODO txid ??

		topupUpdated = update
		return nil
	})

	return topupUpdated, err
}
