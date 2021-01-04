package application

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

// AdminService administrates a TaxiService
type AdminService interface {
	GetFundingAddresses(ctx context.Context, numOfAddresses uint) ([]string, error)
	LatestFundingAddressIndex(ctx context.Context) (int, error)
	ListFundingAddresses(ctx context.Context, start, numOfAddresses uint) ([]domain.AddressInfo, error)
	ListTopups(ctx context.Context, assetHashFilter string) ([]domain.Topup, error)
	AddAsset(ctx context.Context, assetHash string, price decimal.Decimal) error
	Balance(ctx context.Context) (map[string]BalanceInfo, error)
}

type adminService struct {
	hdTreeRepository domain.HDTreeRepository
	topupRepository  domain.TopupRepository
	priceRepository  domain.PriceRepository
	explorerService  explorer.Service
}

// NewAdminService is a factory function for AdminService interface
func NewAdminService(
	hdTreeRepo domain.HDTreeRepository,
	topupRepo domain.TopupRepository,
	priceRepo domain.PriceRepository,
	explorerService explorer.Service,
) AdminService {
	return newAdminService(hdTreeRepo, topupRepo, priceRepo, explorerService)
}
func newAdminService(
	hdTreeRepo domain.HDTreeRepository,
	topupRepo domain.TopupRepository,
	priceRepo domain.PriceRepository,
	explorerService explorer.Service,
) AdminService {
	return &adminService{hdTreeRepo, topupRepo, priceRepo, explorerService}
}

func (adminSvc *adminService) GetFundingAddresses(ctx context.Context, numOfAddresses uint) ([]string, error) {
	if numOfAddresses == 0 {
		numOfAddresses = 1
	}

	return adminSvc.hdTreeRepository.GetNewFundingAddresses(ctx, numOfAddresses)
}

func (adminSvc *adminService) LatestFundingAddressIndex(ctx context.Context) (int, error) {
	hdTree, err := adminSvc.hdTreeRepository.GetOrCreateHDTree(ctx)
	if err != nil {
		return -1, err
	}

	lbtcAccount, ok := hdTree.Accounts[domain.LBTCAccount]
	if !ok {
		return 0, nil
	}

	return int(lbtcAccount.NextExternalIndex) - 1, nil
}

func (adminSvc *adminService) ListFundingAddresses(ctx context.Context, start, numOfAddresses uint) ([]domain.AddressInfo, error) {
	hdTree, err := adminSvc.hdTreeRepository.GetOrCreateHDTree(ctx)
	if err != nil {
		return nil, err
	}

	return hdTree.DeriveAddressInRangeForAccount(domain.LBTCAccount, start, numOfAddresses)
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

func (adminSvc *adminService) Balance(ctx context.Context) (map[string]BalanceInfo, error) {
	hdTree, err := adminSvc.hdTreeRepository.GetOrCreateHDTree(ctx)
	if err != nil {
		return nil, err
	}

	payoutAddresses, err := hdTree.AllDerivedAddressInfoForPayoutAccount()
	if err != nil {
		return nil, err
	}

	fundingAddresses, err := hdTree.AllDerivedAddressInfoForLBTCAccount()
	if err != nil {
		return nil, err
	}

	utxos := make([]explorer.Utxo, 0)

	for _, addressInfo := range append(payoutAddresses, fundingAddresses...) {
		unspents, err := adminSvc.explorerService.GetUnspents(addressInfo.Address, [][]byte{})
		if err != nil {
			return nil, err
		}
		utxos = append(utxos, unspents...)
	}

	balanceInfos := utxosToBalanceInfos(utxos)
	return balanceInfos, nil
}

// BalanceInfo describes balance amounts for an asset
type BalanceInfo struct {
	TotalBalance       uint64
	ConfirmedBalance   uint64
	UnconfirmedBalance uint64
}

func utxosToBalanceInfos(utxos []explorer.Utxo) map[string]BalanceInfo {
	resultMap := make(map[string]BalanceInfo)
	for _, utxo := range utxos {
		value := utxo.Value()
		isConfirmed := utxo.IsConfirmed()

		// asset already known
		if balanceInfo, ok := resultMap[utxo.Asset()]; ok {
			balanceInfo.TotalBalance += value
			if isConfirmed {
				balanceInfo.ConfirmedBalance += value
				continue
			}

			balanceInfo.UnconfirmedBalance += value
			continue
		}

		// new asset = new balance
		balanceInfo := BalanceInfo{
			TotalBalance:       value,
			ConfirmedBalance:   0,
			UnconfirmedBalance: 0,
		}

		if isConfirmed {
			balanceInfo.ConfirmedBalance = value
		} else {
			balanceInfo.UnconfirmedBalance = value
		}

		resultMap[utxo.Asset()] = balanceInfo
	}

	return resultMap
}
