package dbbadger

import (
	"context"
	"path/filepath"

	"github.com/dgraph-io/badger/v2"
	"github.com/timshannon/badgerhold/v2"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

// HDTreeStoreKey is the directory name of the db files
const HDTreeStoreKey = "hdtree"

type badgerHDTreeRepository struct {
	store *badgerhold.Store
}

// NewBadgerHDTreeRepository is a factory function for Badger HDTreeRepository implementation
func NewBadgerHDTreeRepository(baseDirPath string, logger badger.Logger) (domain.HDTreeRepository, error) {
	hdTreeStore, err := createDb(filepath.Join(baseDirPath, HDTreeStoreKey), logger)
	if err != nil {
		return nil, err
	}

	return &badgerHDTreeRepository{
		store: hdTreeStore,
	}, nil
}

func (r *badgerHDTreeRepository) GetOrCreateHDTree(ctx context.Context) (*domain.HDTree, error) {
	return r.getOrCreateHDTree(ctx)
}

func (r *badgerHDTreeRepository) GetNewFundingAddresses(ctx context.Context, numOfAddresses uint) ([]string, error) {
	hdTree, err := r.getOrCreateHDTree(ctx)
	if err != nil {
		return nil, err
	}

	addresses := make([]string, numOfAddresses, numOfAddresses)
	for i := uint(0); i < numOfAddresses; i++ {
		addrInfo, err := hdTree.DeriveNextAddressForLBTCAccount()
		if err != nil {
			return nil, err
		}
		addresses[i] = addrInfo.Address
	}

	if err := r.store.Update(HDTreeStoreKey, hdTree); err != nil {
		return nil, err
	}

	return addresses, nil
}

func (r *badgerHDTreeRepository) GetNewPayoutAddresses(ctx context.Context, numOfAddresses uint) ([]string, error) {
	hdTree, err := r.getOrCreateHDTree(ctx)
	if err != nil {
		return nil, err
	}

	addresses := make([]string, numOfAddresses, numOfAddresses)
	for i := uint(0); i < numOfAddresses; i++ {
		addrInfo, err := hdTree.DeriveNextAddressForPayoutAccount()
		if err != nil {
			return nil, err
		}
		addresses[i] = addrInfo.Address
	}

	if err := r.store.Update(HDTreeStoreKey, hdTree); err != nil {
		return nil, err
	}

	return addresses, nil
}

func (r *badgerHDTreeRepository) getOrCreateHDTree(ctx context.Context) (*domain.HDTree, error) {
	hdTree, err := r.getHDTree(ctx)
	if err != nil {
		return nil, err
	}
	if hdTree != nil {
		return hdTree, nil
	}

	hdTree, err = domain.NewHDTree(wallet.DefaultBaseDerivationPath)
	if err != nil {
		return nil, err
	}

	if err := r.store.Insert(HDTreeStoreKey, hdTree); err != nil {
		return nil, err
	}

	return hdTree, nil
}

func (r *badgerHDTreeRepository) getHDTree(_ context.Context) (*domain.HDTree, error) {
	var hdTree domain.HDTree
	if err := r.store.Get(HDTreeStoreKey, &hdTree); err != nil {
		if err == badgerhold.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &hdTree, nil
}
