package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
)

// AdminService allows to update the list of supported asset for which
// perfoming adding new supported assets for which
type AdminService interface {
	ListTopups(
		ctx context.Context, assetHashFilter string,
	) ([]*domain.Topup, error)
	AddAsset(
		ctx context.Context, assetHash string, price decimal.Decimal,
		basisPoints uint32,
	) error
	AddWhitelistedAddresses(
		ctx context.Context, addresses []string,
	) error
}

type adminService struct {
	wallet      Wallet
	repoManager ports.RepoManager
}

// NewAdminService is a factory function for AdminService interface
func NewAdminService(
	wallet Wallet, repoManager ports.RepoManager,
) AdminService {
	return newAdminService(wallet, repoManager)
}

func newAdminService(
	wallet Wallet, repoManager ports.RepoManager,
) *adminService {
	return &adminService{wallet, repoManager}
}

func (s *adminService) AddAsset(
	ctx context.Context, assetHash string, price decimal.Decimal,
	basisPoints uint32,
) error {
	return s.repoManager.PriceRepository().UpsertPrice(
		ctx, &domain.Price{
			AssetHash:  assetHash,
			Amount:     price,
			BasisPoint: basisPoints,
		},
	)
}

func (s *adminService) AddWhitelistedAddresses(
	ctx context.Context, addresses []string,
) error {
	if _, err := s.repoManager.WhitelistRepository().AddAddresses(
		ctx, addresses,
	); err != nil {
		return err
	}
	return nil
}

func (s *adminService) ListTopups(ctx context.Context, assetHashFilter string) ([]*domain.Topup, error) {
	topups, err := s.repoManager.TopupRepository().GetAllTopups(ctx)
	if err != nil {
		return nil, err
	}

	if assetHashFilter == "" {
		return topups, nil
	}

	filteredTopups := make([]*domain.Topup, 0)
	for _, topup := range topups {
		if topup.AssetHash == assetHashFilter {
			filteredTopups = append(filteredTopups, topup)
		}
	}
	return filteredTopups, nil
}
