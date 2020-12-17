package application

import (
	"context"

	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

// AdminService administrates a TaxiService
type AdminService interface {
	ListTopups(assetHashFilter string) ([]domain.Topup, error)
	ChangeSpread(basisPoint uint32)
}

type adminService struct {
	topupRepository domain.TopupRepository
}

// NewAdminService is a factory function for AdminService interface
func NewAdminService(topupRepository domain.TopupRepository) AdminService {
	return &adminService{
		topupRepository: topupRepository,
	}
}

// ListTopups returns all the topups present in the taxi's topup repository
// optionnaly filtered by assetHash
func (adminSvc *adminService) ListTopups(assetHashFilter string) ([]domain.Topup, error) {
	topups, err := adminSvc.topupRepository.GetAllTopups(context.Background())
	if err != nil {
		return nil, err
	}

	// if no filter: return all the topups
	if assetHashFilter == "" {
		return topups, nil
	}

	// else : filter by assetHash
	filteredTopups := make([]domain.Topup, 0)

	for _, topup := range topups {
		if topup.AssetHash == assetHashFilter {
			filteredTopups = append(filteredTopups, topup)
		}
	}

	return filteredTopups, nil
}

// ChangeSpread sets the basisPoint of the taxiService
func (adminSvc *adminService) ChangeSpread(basisPoint uint32) {
	panic("not implemented")
}
