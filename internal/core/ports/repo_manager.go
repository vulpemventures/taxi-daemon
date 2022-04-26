package ports

import "github.com/vulpemventures/taxi-daemon/internal/core/domain"

type RepoManager interface {
	TopupRepository() domain.TopupRepository
	PriceRepository() domain.PriceRepository
	WhitelistRepository() domain.WhitelistRepository
	Close()
}
