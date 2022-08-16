package inmemory

import (
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
)

type repoManager struct {
	priceRepository     *priceRepository
	topupRepository     *topupRepository
	whitelistRepository *whitelistRepository
}

func NewRepoManager() ports.RepoManager {
	priceRepo := newPriceRepository()
	topupRepo := newTopupRepository()
	whitelistRepo := newWhitelistRepository()

	return &repoManager{
		priceRepository:     priceRepo,
		topupRepository:     topupRepo,
		whitelistRepository: whitelistRepo,
	}
}

func (rm *repoManager) PriceRepository() domain.PriceRepository {
	return rm.priceRepository
}

func (rm *repoManager) TopupRepository() domain.TopupRepository {
	return rm.topupRepository
}

func (rm *repoManager) WhitelistRepository() domain.WhitelistRepository {
	return rm.whitelistRepository
}

func (rm *repoManager) Close() {}
