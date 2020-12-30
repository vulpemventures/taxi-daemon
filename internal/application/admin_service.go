package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

// AdminService administrates a TaxiService
type AdminService interface {
	GetFundingAddresses(ctx context.Context, numOfAddresses uint) ([]string, error)
	LastFundingAddressIndex(ctx context.Context) (int, error)
	ListFundingAddresses(ctx context.Context, start, numOfAddresses uint) ([]string, error)
	ListTopups(ctx context.Context, assetHashFilter string) ([]domain.Topup, error)
	AddAsset(ctx context.Context, assetHash string, price decimal.Decimal) error
	ChangeSpread(ctx context.Context, basisPoint uint32)
}

type adminService struct {
	hdTreeRepository domain.HDTreeRepository
	topupRepository  domain.TopupRepository
	priceRepository  domain.PriceRepository
}

// NewAdminService is a factory function for AdminService interface
func NewAdminService(
	hdTreeRepo domain.HDTreeRepository,
	topupRepo domain.TopupRepository,
	priceRepo domain.PriceRepository,
) AdminService {
	return newAdminService(hdTreeRepo, topupRepo, priceRepo)
}
func newAdminService(
	hdTreeRepo domain.HDTreeRepository,
	topupRepo domain.TopupRepository,
	priceRepo domain.PriceRepository,
) AdminService {
	return &adminService{hdTreeRepo, topupRepo, priceRepo}
}

func (adminSvc *adminService) GetFundingAddresses(ctx context.Context, numOfAddresses uint) ([]string, error) {
	if numOfAddresses == 0 {
		numOfAddresses = 1
	}

	return adminSvc.hdTreeRepository.GetNewFundingAddresses(ctx, numOfAddresses)
}

func (adminSvc *adminService) LastFundingAddressIndex(ctx context.Context) (int, error) {
	hdTree, err := adminSvc.hdTreeRepository.GetOrCreateHDTree(ctx)
	if err != nil {
		return -1, err
	}

	lastIndex := int(hdTree.Accounts[domain.LBTCAccount].NextExternalIndex) - 1
	return lastIndex, nil
}

func (adminSvc *adminService) ListFundingAddresses(ctx context.Context, start, numOfAddresses uint) ([]string, error) {
	hdTree, err := adminSvc.hdTreeRepository.GetOrCreateHDTree(ctx)
	if err != nil {
		return nil, err
	}

	infoList, err := hdTree.DeriveAddressInRangeForAccount(domain.LBTCAccount, start, numOfAddresses)
	if err != nil {
		return nil, err
	}

	addresses := make([]string, len(infoList), len(infoList))
	for i, info := range infoList {
		addresses[i] = info.Address
	}
	return addresses, nil
}

func (adminSvc *adminService) AddAsset(ctx context.Context, assetHash string, price decimal.Decimal) error {
	return adminSvc.priceRepository.UpdateOrCreatePrice(ctx, domain.Price{
		AssetHash: assetHash,
		Amount:    price,
	})
}

// ListTopups returns all the topups present in the taxi's topup repository
// optionnaly filtered by assetHash
func (adminSvc *adminService) ListTopups(ctx context.Context, assetHashFilter string) ([]domain.Topup, error) {
	topups, err := adminSvc.topupRepository.GetAllTopups(ctx)
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
func (adminSvc *adminService) ChangeSpread(_ context.Context, basisPoint uint32) {
	panic("not implemented")
}
