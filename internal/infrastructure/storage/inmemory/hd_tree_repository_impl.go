package inmemory

import (
	"context"
	"strings"
	"sync"

	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

const (
	sentence = "clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act soccer drive year mask"
)

var (
	words = strings.Split(sentence, " ")
)

type inmemoryHDTreeRepository struct {
	hdTree   *domain.HDTree
	locker   sync.Locker
	network  network.Network
	mnemonic []string
}

// NewInmemoryHDTreeRepository creates an inmemory implementatoin of the
// HDTreeRepository interface
func NewInmemoryHDTreeRepository(network network.Network, mnemonic []string) domain.HDTreeRepository {
	return newInmemoryHDTreeRepository(network, mnemonic)
}

func newInmemoryHDTreeRepository(network network.Network, mnemonic []string) *inmemoryHDTreeRepository {
	return &inmemoryHDTreeRepository{
		locker:   &sync.Mutex{},
		network:  network,
		mnemonic: mnemonic,
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
		hdTree, err := domain.NewHDTree(r.mnemonic, wallet.DefaultBaseDerivationPath, r.network)
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
