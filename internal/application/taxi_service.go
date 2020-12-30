package application

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/shopspring/decimal"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

const (
	derivationPathForAsset = "0'/0/0"
	derivationPathForLBTC  = "0'/0/1"
)

// TaxiService provides interface for user UI
type TaxiService interface {
	TopupWithAsset(ctx context.Context, assetHash string) (*domain.Topup, error)
	ListAssets(ctx context.Context) ([]string, error)
}

type taxiService struct {
	hdTreeRepository domain.HDTreeRepository
	lockRepository   domain.LockRepository
	topupRepository  domain.TopupRepository
	priceRepository  domain.PriceRepository
	explorer         explorer.Service
}

// NewTaxiService returns a TaxiService interface
func NewTaxiService(
	hdTreeRepo domain.HDTreeRepository,
	lockRepo domain.LockRepository,
	topupRepo domain.TopupRepository,
	priceRepo domain.PriceRepository,
	explorerSvc explorer.Service,
) TaxiService {
	return newTaxiService(hdTreeRepo, lockRepo, topupRepo, priceRepo, explorerSvc)
}

func newTaxiService(
	hdTreeRepo domain.HDTreeRepository,
	lockRepo domain.LockRepository,
	topupRepo domain.TopupRepository,
	priceRepo domain.PriceRepository,
	explorerSvc explorer.Service,
) *taxiService {
	return &taxiService{hdTreeRepo, lockRepo, topupRepo, priceRepo, explorerSvc}
}

// TopupWithAsset returns a transaction letting to pay a given amount of fees with assetHash token.
func (taxi *taxiService) TopupWithAsset(ctx context.Context, assetHash string) (*domain.Topup, error) {
	utxoTTL := time.Duration(config.GetInt(config.UtxoTTLKey)) * time.Second
	// check if asset is available
	price, err := taxi.priceRepository.GetPrice(ctx, assetHash)
	if err != nil {
		return nil, err
	}
	if price == nil {
		return nil, ErrAssetNotSupported{AssetHash: assetHash}
	}

	// retrieve only the utxos that are not locked by other topups
	utxos, err := taxi.fetchAvailableUtxos(ctx)
	if err != nil {
		return nil, err
	}

	feeAmount := uint64(config.GetInt(config.FixedFeeAmountKey))
	lbtcAsset := config.GetNetwork().AssetID

	utxo, err := selectOneUtxo(utxos, feeAmount, lbtcAsset)
	if err != nil {
		return nil, err
	}

	expireAt := time.Now().Add(utxoTTL)

	topup := domain.NewTopup(assetHash, feeAmount, price.Amount, expireAt)
	err = taxi.topupRepository.AddTopup(ctx, topup)
	if err != nil {
		return nil, err
	}

	amountAsDecimal := decimal.NewFromInt(int64(feeAmount))
	assetAmount := uint64(price.Amount.Mul(amountAsDecimal).IntPart())

	payouts, err := taxi.hdTreeRepository.GetNewPayoutAddresses(ctx, 1)
	if err != nil {
		return nil, err
	}

	psetBase64, err := taxi.createAndSignTopupTx(createAndSignTopupTxArgs{
		unspent:       utxo,
		assetHash:     assetHash,
		assetAmount:   assetAmount,
		payoutAddress: payouts[0],
	})
	if err != nil {
		return nil, err
	}

	topupCompleted, err := taxi.topupRepository.MarkAsCompleted(ctx, topup.ID, psetBase64, assetAmount)
	if err != nil {
		return nil, err
	}

	// Lock the utxo used for this topup to avoid other ones to use it
	utxoKey := domain.UtxoToLock(utxo)
	if err = taxi.lockRepository.Lock(ctx, []domain.Lock{utxoKey}); err != nil {
		return nil, err
	}

	go func() {
		time.Sleep(utxoTTL)
		taxi.lockRepository.Unlock(ctx, utxoKey)
	}()

	return topupCompleted, nil
}

// ListAssets return all the available assets of the taxi service
func (taxi *taxiService) ListAssets(ctx context.Context) ([]string, error) {
	prices, err := taxi.priceRepository.GetAllPrices(ctx)
	if err != nil {
		return nil, err
	}

	assets := make([]string, 0)
	for _, price := range prices {
		assets = append(assets, price.AssetHash)
	}

	return assets, nil
}

func (taxi *taxiService) fetchAvailableUtxos(ctx context.Context) ([]explorer.Utxo, error) {
	// retrieve root path of the hd tree and instantiate the inmemory wallet
	hdTree, err := taxi.hdTreeRepository.GetOrCreateHDTree(ctx)
	if err != nil {
		return nil, err
	}

	// get all funding derived addresses
	addrInfoList, err := hdTree.AllDerivedAddressInfoForLBTCAccount()
	if err != nil {
		return nil, err
	}

	availableUtxos := make([]explorer.Utxo, 0)
	for _, addrInfo := range addrInfoList {
		utxos, err := taxi.explorer.GetUnspents(addrInfo.Address, [][]byte{})
		if err != nil {
			return nil, err
		}

		for _, utxo := range utxos {
			utxoKey := domain.UtxoToLock(utxo)
			if !taxi.lockRepository.IsLocked(ctx, utxoKey) {
				availableUtxos = append(availableUtxos, utxo)
			}
		}
	}

	return availableUtxos, nil
}

func selectOneUtxo(utxos []explorer.Utxo, targetAmount uint64, targetAsset string) (explorer.Utxo, error) {
	for _, utxo := range utxos {
		if !utxo.IsConfidential() && utxo.Asset() == targetAsset && utxo.Value() == targetAmount {
			return utxo, nil
		}
	}

	return nil, ErrUtxoNotFound{
		TargetedAmount: targetAmount,
		TargetedAsset:  targetAsset,
	}
}

type createAndSignTopupTxArgs struct {
	context       context.Context
	unspent       explorer.Utxo
	assetHash     string
	assetAmount   uint64
	payoutAddress string
}

func (taxi *taxiService) createAndSignTopupTx(args createAndSignTopupTxArgs) (string, error) {
	hdTree, err := taxi.hdTreeRepository.GetOrCreateHDTree(args.context)
	if err != nil {
		return "", err
	}
	rootPath, _ := wallet.ParseDerivationPath(hdTree.RootDerivationPath)
	w, err := wallet.NewWallet(config.GetMnemonic(), rootPath, *config.GetNetwork())
	if err != nil {
		return "", err
	}

	outputsOpts := []wallet.Receiver{
		{
			Address: args.payoutAddress,
			Asset:   args.assetHash,
			Amount:  args.assetAmount,
		},
		{
			Asset:  config.GetNetwork().AssetID,
			Amount: uint64(config.GetInt(config.FixedFeeAmountKey)),
		},
	}
	psetBase64, err := w.CreatePSET(
		[]explorer.Utxo{args.unspent},
		outputsOpts,
	)
	if err != nil {
		return "", err
	}

	derivationPath, _ := hdTree.DerivationPathForScript(
		hex.EncodeToString(args.unspent.Script()),
	)

	signOpts := wallet.SignInputArgs{
		PsetBase64: psetBase64,
		Path:       derivationPath,
		InputIndex: 0,
		HashType:   txscript.SigHashSingle | txscript.SigHashAnyOneCanPay,
	}

	return w.SignInput(signOpts)
}
