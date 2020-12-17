package dbbadger

import (
	"context"
	"path/filepath"

	"github.com/dgraph-io/badger/v2"
	"github.com/timshannon/badgerhold/v2"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

const TopupStoreKey = "topup"

type badgerTopupRepository struct {
	store *badgerhold.Store
}

func NewBadgerTopupRepository(baseDirPath string, logger badger.Logger) (domain.TopupRepository, error) {
	topupStore, err := createDb(filepath.Join(baseDirPath, TopupStoreKey), logger)
	if err != nil {
		return nil, err
	}

	return &badgerTopupRepository{
		store: topupStore,
	}, nil
}

func (r *badgerTopupRepository) AddTopup(ctx context.Context, topup Topup) error {

}

func (r *badgerTopupRepository) GetAllTopups(ctx context.Context) ([]Topup, error) {

}

func (r *badgerTopupRepository) SetRequested(ctx context.Context, topupId string) (*Topup, error) {}

func (r *badgerTopupRepository)

func (r *badgerTopupRepository)