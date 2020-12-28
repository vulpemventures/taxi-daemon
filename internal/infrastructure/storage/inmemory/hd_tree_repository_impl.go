package inmemory

import (
	"context"
	"sync"

	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

type inmemoryHDTreeRepository struct {
	hdTree *domain.HDTree
	locker sync.Locker
}

// NewInmemoryHDTreeRepository creates an inmemory implementatoin of the
// HDTreeRepository interface
func NewInmemoryHDTreeRepository() domain.HDTreeRepository {
	return newInmemoryHDTreeRepository()
}

func newInmemoryHDTreeRepository() *inmemoryHDTreeRepository {
	return &inmemoryHDTreeRepository{
		locker: &sync.Mutex{},
	}
}

func (r *inmemoryHDTreeRepository) GetOrCreateHDTree(ctx context.Context) (*domain.HDTree, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

	return r.getOrCreateHDTree(ctx)
}

func (r *inmemoryHDTreeRepository) GetNewFundingAddresses(ctx context.Context, numOfAddresses uint) ([]string, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

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

	r.hdTree = hdTree

	return addresses, nil
}

func (r *inmemoryHDTreeRepository) GetNewPayoutAddresses(ctx context.Context, numOfAddresses uint) ([]string, error) {
	r.locker.Lock()
	defer r.locker.Unlock()

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

	r.hdTree = hdTree

	return addresses, nil
}

func (r *inmemoryHDTreeRepository) getOrCreateHDTree(_ context.Context) (*domain.HDTree, error) {
	if r.hdTree == nil {
		hdTree, err := domain.NewHDTree(wallet.DefaultBaseDerivationPath)
		if err != nil {
			return nil, err
		}
		r.hdTree = hdTree
	}

	return &domain.HDTree{
		RootDerivationPath: r.hdTree.RootDerivationPath,
		Accounts:           r.hdTree.Accounts,
	}, nil
}
