package dbbadger

import (
	"context"
	"path/filepath"

	"github.com/dgraph-io/badger/v2"
	"github.com/timshannon/badgerhold/v2"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/taxi-daemon/internal/domain"
	"github.com/vulpemventures/taxi-daemon/pkg/wallet"
)

// HDTreeStoreKey is the directory name of the db files
const HDTreeStoreKey = "hdtree"

type badgerHDTreeRepository struct {
	store    *badgerhold.Store
	network  network.Network
	mnemonic []string
}

// NewBadgerHDTreeRepository is a factory function for Badger HDTreeRepository implementation
func NewBadgerHDTreeRepository(baseDirPath string, mnemonic []string, network network.Network, logger badger.Logger) (domain.HDTreeRepository, error) {
	hdTreeStore, err := createDb(filepath.Join(baseDirPath, HDTreeStoreKey), logger)
	if err != nil {
		return nil, err
	}

	return &badgerHDTreeRepository{
		store:    hdTreeStore,
		network:  network,
		mnemonic: mnemonic,
	}, nil
}

func (r *badgerHDTreeRepository) GetOrCreateHDTree(ctx context.Context) (*domain.HDTree, error) {
	return r.getOrCreateHDTree(ctx)
}

func (r *badgerHDTreeRepository) GetNewFundingAddresses(ctx context.Context, numOfAddresses uint, isChange bool) ([]string, error) {
	hdTree, err := r.getOrCreateHDTree(ctx)
	if err != nil {
		return nil, err
	}

	addresses := make([]string, numOfAddresses, numOfAddresses)
	for i := uint(0); i < numOfAddresses; i++ {
		addrInfo, err := hdTree.DeriveNextAddressForLBTCAccount(isChange)
		if err != nil {
			return nil, err
		}
		addresses[i] = addrInfo.Address
	}

	if err := r.update(hdTree); err != nil {
		return nil, err
	}

	return addresses, nil
}

func (r *badgerHDTreeRepository) GetNewPayoutAddresses(ctx context.Context, numOfAddresses uint, isChange bool) ([]string, error) {
	hdTree, err := r.getOrCreateHDTree(ctx)
	if err != nil {
		return nil, err
	}

	addresses := make([]string, numOfAddresses, numOfAddresses)
	for i := uint(0); i < numOfAddresses; i++ {
		addrInfo, err := hdTree.DeriveNextAddressForPayoutAccount(isChange)
		if err != nil {
			return nil, err
		}
		addresses[i] = addrInfo.Address
	}

	err = r.update(hdTree)
	if err != nil {
		return nil, err
	}
	return addresses, nil
}

func (r *badgerHDTreeRepository) update(hdTree *domain.HDTree) error {
	handler := func() (interface{}, error) {
		if err := r.store.Update(HDTreeStoreKey, hdTree); err != nil {
			return nil, err
		}

		return nil, nil
	}

	_, err := runTransaction(handler)
	return err
}

func (r *badgerHDTreeRepository) getOrCreateHDTree(ctx context.Context) (*domain.HDTree, error) {
	hdTree, err := r.getHDTree(ctx)
	if err != nil {
		return nil, err
	}
	if hdTree != nil {
		return hdTree, nil
	}

	hdTree, err = domain.NewHDTree(r.mnemonic, wallet.DefaultBaseDerivationPath, r.network)
	if err != nil {
		return nil, err
	}

	handler := func() (interface{}, error) {
		if err := r.store.Insert(HDTreeStoreKey, hdTree); err != nil {
			return nil, err
		}
		return nil, nil
	}

	_, err = runTransaction(handler)
	if err != nil {
		return nil, err
	}

	return hdTree, nil
}

func (r *badgerHDTreeRepository) getHDTree(_ context.Context) (*domain.HDTree, error) {
	handler := func() (interface{}, error) {
		var hdTree domain.HDTree
		if err := r.store.Get(HDTreeStoreKey, &hdTree); err != nil {
			return nil, err
		}
		return hdTree, nil
	}

	res, err := runTransaction(handler)
	if err != nil {
		if err == badgerhold.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	hdTree := res.(domain.HDTree)
	return &hdTree, nil
}
