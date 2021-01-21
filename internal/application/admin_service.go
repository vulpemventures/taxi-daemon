package application

import (
	"context"
	"encoding/hex"

	"github.com/btcsuite/btcd/txscript"
	"github.com/shopspring/decimal"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/taxi-daemon/internal/domain"
	"github.com/vulpemventures/taxi-daemon/pkg/fee"
	"github.com/vulpemventures/taxi-daemon/pkg/wallet"
)

// AdminService administrates a TaxiService
type AdminService interface {
	GetFundingAddresses(ctx context.Context, numOfAddresses uint) ([]string, error)
	LatestFundingAddressIndex(ctx context.Context) (int, error)
	ListFundingAddresses(ctx context.Context, start, numOfAddresses uint, change bool) ([]domain.AddressInfo, error)
	ListTopups(ctx context.Context, assetHashFilter string) ([]domain.Topup, error)
	AddAsset(ctx context.Context, assetHash string, price decimal.Decimal, spread decimal.Decimal) error
	SendToMany(ctx context.Context, outputs []wallet.Receiver, millisatPerByte int64, pushTransaction bool) (string, error)
	Balance(ctx context.Context) (map[string]*BalanceInfo, error)
	ListLocks(ctx context.Context) ([]domain.Lock, error)
}

type adminService struct {
	hdTreeRepository domain.HDTreeRepository
	topupRepository  domain.TopupRepository
	priceRepository  domain.PriceRepository
	lockRepository   domain.LockRepository
	cache            FragmentCache
	explorerService  explorer.Service
	mnemonic         []string
	network          network.Network
}

// NewAdminServiceArgs ...
type NewAdminServiceArgs struct {
	HDTreeRepository domain.HDTreeRepository
	TopupRepository  domain.TopupRepository
	PriceRepository  domain.PriceRepository
	LockRepository   domain.LockRepository
	ExplorerService  explorer.Service
	Mnemonic         []string
	Network          network.Network
	Cache            FragmentCache
}

// NewAdminService is a factory function for AdminService interface
func NewAdminService(args NewAdminServiceArgs) AdminService {
	return newAdminService(args)
}
func newAdminService(args NewAdminServiceArgs) *adminService {
	return &adminService{
		hdTreeRepository: args.HDTreeRepository,
		topupRepository:  args.TopupRepository,
		priceRepository:  args.PriceRepository,
		lockRepository:   args.LockRepository,
		cache:            args.Cache,
		explorerService:  args.ExplorerService,
		mnemonic:         args.Mnemonic,
		network:          args.Network,
	}
}

func (adminSvc *adminService) ListLocks(ctx context.Context) ([]domain.Lock, error) {
	return adminSvc.lockRepository.GetLocks(ctx)
}

func (adminSvc *adminService) SendToMany(ctx context.Context, outputs []wallet.Receiver, millisatPerByte int64, pushTransaction bool) (string, error) {
	hdTree, err := adminSvc.hdTreeRepository.GetOrCreateHDTree(ctx)
	if err != nil {
		return "", err
	}
	rootPath, _ := wallet.ParseDerivationPath(hdTree.RootDerivationPath)

	w, err := wallet.NewWallet(adminSvc.mnemonic, rootPath, adminSvc.network)
	if err != nil {
		return "", err
	}

	// first coin selection for fees estimation
	selectedUtxos, changeOutputs, err := adminSvc.selectUtxos(ctx, outputs, w)
	if err != nil {
		return "", err
	}

	feeAmount := fee.EstimateTxFee(
		len(selectedUtxos),
		len(outputs)+len(changeOutputs),
		true,
		int(millisatPerByte),
	)

	feeOutput := wallet.Receiver{
		Asset:  w.GetNetwork().AssetID,
		Amount: feeAmount,
	}

	// redo the coinselection
	selectedUtxos, changeOutputs, err = adminSvc.selectUtxos(ctx, append(outputs, feeOutput), w)
	if err != nil {
		return "", err
	}

	outputs = append(outputs, changeOutputs...)
	outputs = append(outputs, feeOutput)

	base64PSET, err := w.CreatePSET(
		selectedUtxos,
		outputs,
	)
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

		base64PSET, err = w.SignInput(signOpts)
		if err != nil {
			return "", err
		}
	}

	hex, err := w.FinalizeAndExtract(base64PSET)
	if err != nil {
		return "", err
	}

	if pushTransaction {
		_, err := adminSvc.explorerService.BroadcastTransaction(hex)
		if err != nil {
			return "", err
		}
	}

	return hex, nil
}

func (adminSvc *adminService) GetFundingAddresses(ctx context.Context, numOfAddresses uint) ([]string, error) {
	if numOfAddresses == 0 {
		numOfAddresses = 1
	}

	return adminSvc.hdTreeRepository.GetNewFundingAddresses(ctx, numOfAddresses, false)
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

func (adminSvc *adminService) ListFundingAddresses(ctx context.Context, start, numOfAddresses uint, change bool) ([]domain.AddressInfo, error) {
	hdTree, err := adminSvc.hdTreeRepository.GetOrCreateHDTree(ctx)
	if err != nil {
		return nil, err
	}

	return hdTree.DeriveAddressInRangeForAccount(domain.LBTCAccount, start, numOfAddresses, change)
}

func (adminSvc *adminService) AddAsset(ctx context.Context, assetHash string, price decimal.Decimal, spread decimal.Decimal) error {
	return adminSvc.priceRepository.UpdateOrCreatePrice(ctx, domain.Price{
		AssetHash: assetHash,
		Amount:    price,
		Spread:    spread,
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

// BalanceInfo describes balance amounts for an asset
type BalanceInfo struct {
	TotalBalance       uint64
	ConfirmedBalance   uint64
	UnconfirmedBalance uint64
}

func (adminSvc *adminService) Balance(ctx context.Context) (map[string]*BalanceInfo, error) {
	utxos, err := adminSvc.getUnspents(ctx)
	if err != nil {
		return nil, err
	}

	balanceInfos := utxosToBalanceInfos(utxos)
	return balanceInfos, nil
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

func utxosToBalanceInfos(utxos []explorer.Utxo) map[string]*BalanceInfo {
	resultMap := make(map[string]*BalanceInfo)
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
		balanceInfo := &BalanceInfo{
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

func (adminSvc *adminService) selectUtxos(ctx context.Context, outputs []wallet.Receiver, w wallet.Wallet) ([]explorer.Utxo, []wallet.Receiver, error) {
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
				adminSvc.cache.Delete(Outpoint{
					TxID: utxo.Hash(),
					Vout: utxo.Index(),
				})

				selectedUtxos = append(selectedUtxos, utxo)
				totalSelected += utxo.Value()
				if totalSelected >= amount {
					changeMap[asset] += totalSelected - amount
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

			if asset == w.GetNetwork().AssetID {
				addrs, err := adminSvc.hdTreeRepository.GetNewFundingAddresses(ctx, 1, true)
				if err != nil {
					return nil, nil, err
				}
				address = addrs[0]
			} else {
				addrs, err := adminSvc.hdTreeRepository.GetNewPayoutAddresses(ctx, 1, true)
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
