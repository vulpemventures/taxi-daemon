package dbbadger

import (
	"context"
	"errors"
	"path/filepath"

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

func (r *badgerTopupRepository) MarkAsCompleted(ctx context.Context, opts domain.MarkAsCompletedOpts) (*domain.Topup, error) {
	var topupUpdated *domain.Topup

	query := badgerhold.Where("ID").Eq(opts.TopupID)
	err := r.store.UpdateMatching(&domain.Topup{}, query, func(record interface{}) error {
		update, ok := record.(*domain.Topup)
		if !ok {
			return errors.New("record isn't the correct type! Need domain.Topup type")
		}

		update.Complete(opts.Partial, opts.AssetAmount, opts.BlindingPrivateKey, opts.BlindingPublicKey)

		topupUpdated = update
		return nil
	})

	return topupUpdated, err
}

func (r *badgerTopupRepository) MarkAsSettled(ctx context.Context, topupID string, txID string) (*domain.Topup, error) {
	var topupUpdated *domain.Topup

	query := badgerhold.Where("ID").Eq(topupID)
	err := r.store.UpdateMatching(&domain.Topup{}, query, func(record interface{}) error {
		update, ok := record.(*domain.Topup)
		if !ok {
			return errors.New("record isn't the correct type! Need domain.Topup type")
		}

		update.Settle()
		// TODO txid ??

		topupUpdated = update
		return nil
	})

	return topupUpdated, err
}
