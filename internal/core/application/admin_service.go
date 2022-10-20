package application

import (
	"context"
	"encoding/hex"
	"github.com/vulpemventures/go-elements/address"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
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
	) (err error)
	AddWhitelistedAddresses(
		ctx context.Context, addresses []string,
	) (err error)
	ListWhitelistedScripts(
		ctx context.Context,
	) (whitelist []string, err error)
	DeleteWhitelistedAddresses(
		ctx context.Context, addresses []string,
	) (err error)
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
	scripts, err := addressesToScripts(addresses)
	if err != nil {
		return err
	}

	count, err := s.repoManager.WhitelistRepository().AddScripts(ctx, scripts)
	if err != nil {
		return err
	}
	if count > 0 {
		log.Debugf("added %d addresses to whitelist", count)
	}
	return nil
}

func (s *adminService) ListWhitelistedScripts(
	ctx context.Context,
) ([]string, error) {
	scripts, err := s.repoManager.WhitelistRepository().GetScripts(ctx)
	if err != nil {
		return nil, err
	}
	return scripts, nil
}

func (s *adminService) DeleteWhitelistedAddresses(
	ctx context.Context, addresses []string,
) error {
	scripts, err := addressesToScripts(addresses)
	if err != nil {
		return err
	}

	count, err := s.repoManager.WhitelistRepository().RemoveScripts(
		ctx, scripts,
	)
	if err != nil {
		return err
	}
	if count > 0 {
		log.Debugf("removed %d addresses from whitelist", count)
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

func addressesToScripts(addresses []string) ([]string, error) {
	scripts := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		script, err := address.ToOutputScript(addr)
		if err != nil {
			return nil, err
		}

		scripts = append(scripts, hex.EncodeToString(script))
	}
	return scripts, nil
}
