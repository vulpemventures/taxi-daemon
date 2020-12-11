package application

import "github.com/vulpemventures/liquid-taxi/internal/domain"

type AdminService interface {
	ListTopups(assetHashFilter string) []domain.Topup
	ChangeSpread(basisPoint int32)
}

// TO DO implementation
