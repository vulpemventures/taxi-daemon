package application

import (
	"context"
	"encoding/hex"

	"github.com/btcsuite/btcd/txscript"
	"github.com/shopspring/decimal"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

// AdminService administrates a TaxiService
type AdminService interface {
	GetFundingAddresses(ctx context.Context, numOfAddresses uint) ([]string, error)
	LatestFundingAddressIndex(ctx context.Context) (int, error)
	ListFundingAddresses(ctx context.Context, start, numOfAddresses uint) ([]domain.AddressInfo, error)
	ListTopups(ctx context.Context, assetHashFilter string) ([]domain.Topup, error)
	AddAsset(ctx context.Context, assetHash string, price decimal.Decimal) error
	SendToMany(ctx context.Context, outputs []wallet.Receiver, millisatPerByte int64, pushTransaction bool) (string, error)
}

type adminService struct {
	wallet           wallet.Wallet
	hdTreeRepository domain.HDTreeRepository
	topupRepository  domain.TopupRepository
	priceRepository  domain.PriceRepository
	explorerService  explorer.Service
}

// NewAdminService is a factory function for AdminService interface
func NewAdminService(
	wallet wallet.Wallet,
	hdTreeRepo domain.HDTreeRepository,
	topupRepo domain.TopupRepository,
	priceRepo domain.PriceRepository,
	explorerService explorer.Service,
) AdminService {
	return newAdminService(wallet, hdTreeRepo, topupRepo, priceRepo, explorerService)
}
func newAdminService(
	wallet wallet.Wallet,
	hdTreeRepo domain.HDTreeRepository,
	topupRepo domain.TopupRepository,
	priceRepo domain.PriceRepository,
	explorerService explorer.Service,
) AdminService {
	return &adminService{wallet, hdTreeRepo, topupRepo, priceRepo, explorerService}
}

func (adminSvc *adminService) SendToMany(ctx context.Context, outputs []wallet.Receiver, millisatPerByte int64, pushTransaction bool) (string, error) {
	selectedUtxos, changeOutputs, err := adminSvc.selectUtxos(ctx, outputs)
	if err != nil {
		return "", err
	}

	base64PSET, err := adminSvc.wallet.CreatePSET(
		selectedUtxos,
		append(outputs, changeOutputs...),
	)
	if err != nil {
		return "", err
	}

	base64PSET, err = adminSvc.wallet.FundPSET(
		base64PSET,
		millisatPerByte,
	)

	hdTree, err := adminSvc.hdTreeRepository.GetOrCreateHDTree(ctx)
	if err != nil {
		return "", err
	}

	for inputIndex, selectedUtxo := range selectedUtxos {
		derivationPath, _ := hdTree.DerivationPathForScript(
			hex.EncodeToString(selectedUtxo.Script()),
		)

		signOpts := wallet.SignInputArgs{
			PsetBase64: base64PSET,
			Path:       derivationPath,
			InputIndex: inputIndex,
			HashType:   txscript.SigHashAll,
		}

		base64PSET, err = adminSvc.wallet.SignInput(signOpts)
		if err != nil {
			return "", err
		}
	}

	return adminSvc.wallet.FinalizeAndExtract(base64PSET)
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

	lastIndex := int(hdTree.Accounts[domain.LBTCAccount].NextExternalIndex) - 1
	return lastIndex, nil
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

func (adminSvc *adminService) getUnspents(ctx context.Context) ([]explorer.Utxo, error) {
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

	return utxos, nil
}

func (adminSvc *adminService) selectUtxos(ctx context.Context, outputs []wallet.Receiver) ([]explorer.Utxo, []wallet.Receiver, error) {
	utxos, err := adminSvc.getUnspents(ctx)
	if err != nil {
		return nil, nil, err
	}

	coinNeedMap := make(map[string]uint64)
	for _, output := range outputs {
		_, ok := coinNeedMap[output.Asset]
		if !ok {
			coinNeedMap[output.Asset] = output.Amount
			continue
		}

		coinNeedMap[output.Asset] += output.Amount
	}

	selectedUtxos := make([]explorer.Utxo, 0)
	changeMap := make(map[string]uint64)

	for asset, amount := range coinNeedMap {
		totalSelected := uint64(0)

		for _, utxo := range utxos {
			if utxo.Asset() == asset && !utxo.IsConfidential() {
				selectedUtxos = append(selectedUtxos, utxo)
				totalSelected += utxo.Value()
				if totalSelected >= amount {
					changeMap[asset] = amount - totalSelected
					break
				}
			}
		}

		if totalSelected < amount {
			return nil, nil, ErrCoinSelection{amount, asset}
		}
	}

	changeOutputs := make([]wallet.Receiver, 0, len(changeMap))
	for asset, changeAmount := range changeMap {
		if changeAmount > 0 {
			var address string

			if asset == adminSvc.wallet.GetNetwork().AssetID {
				addrs, err := adminSvc.hdTreeRepository.GetNewFundingAddresses(ctx, 1)
				if err != nil {
					return nil, nil, err
				}
				address = addrs[0]
			} else {
				addrs, err := adminSvc.hdTreeRepository.GetNewPayoutAddresses(ctx, 1)
				if err != nil {
					return nil, nil, err
				}
				address = addrs[0]
			}

			changeOutputs = append(changeOutputs, wallet.Receiver{
				Address: address,
				Asset:   asset,
				Amount:  changeAmount,
			})
		}
	}

	return selectedUtxos, changeOutputs, nil
}
