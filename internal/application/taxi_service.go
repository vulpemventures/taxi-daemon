package application

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/shopspring/decimal"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/go-elements/network"
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
	network          network.Network
	fixedFeeAmount   uint64
	mnemonic         []string
	cache            FragmentCache
}

// NewTaxiServiceArgs is a struct defining the arguments used to create a taxi service
type NewTaxiServiceArgs struct {
	HdTreeRepo     domain.HDTreeRepository
	LockRepo       domain.LockRepository
	TopupRepo      domain.TopupRepository
	PriceRepo      domain.PriceRepository
	Network        network.Network
	FixedFeeAmount uint64
	Mnemonic       []string
	Cache          FragmentCache
}

// NewTaxiService returns a TaxiService interface
func NewTaxiService(args NewTaxiServiceArgs) TaxiService {
	return newTaxiService(args)
}

func newTaxiService(args NewTaxiServiceArgs) *taxiService {
	taxiSvc := &taxiService{
		args.HdTreeRepo,
		args.LockRepo,
		args.TopupRepo,
		args.PriceRepo,
		args.Network,
		args.FixedFeeAmount,
		args.Mnemonic,
		args.Cache,
	}
	// start caching
	go taxiSvc.cache.Start()

	return taxiSvc
}

// TopupWithAsset returns a transaction letting to pay a given amount of fees with assetHash token.
func (taxi *taxiService) TopupWithAsset(ctx context.Context, assetHash string) (*domain.Topup, error) {
	// check if asset is available
	price, err := taxi.priceRepository.GetPrice(ctx, assetHash)
	if err != nil {
		return nil, err
	}

	if price == nil {
		assets, _ := taxi.ListAssets(ctx)
		return nil, ErrAssetNotSupported{
			AssetHash:     assetHash,
			AssetsAllowed: assets,
		}
	}

	// retrieve only the utxos that are not locked by other topups
	utxo, err := taxi.fetchAndLockFragmentUtxo(ctx)
	if err != nil {
		return nil, err
	}

	expireAt := time.Now().Add(taxi.lockRepository.GetConfigTTL(ctx))

	topup := domain.NewTopup(assetHash, taxi.fixedFeeAmount, price.Amount, expireAt)
	err = taxi.topupRepository.AddTopup(ctx, topup)
	if err != nil {
		return nil, err
	}

	amountAsDecimal := decimal.NewFromInt(int64(taxi.fixedFeeAmount))
	assetAmount := uint64(price.Amount.Mul(amountAsDecimal).IntPart())

	payouts, err := taxi.hdTreeRepository.GetNewPayoutAddresses(ctx, 1, false)
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

func (taxi *taxiService) fetchAndLockFragmentUtxo(ctx context.Context) (explorer.Utxo, error) {
	utxo, err := taxi.cache.RandomGetAndDelete()
	if err != nil {
		return nil, err
	}

	if utxo == nil {
		return nil, ErrUtxoNotFound{
			TargetedAmount: taxi.fixedFeeAmount,
			TargetedAsset:  taxi.network.AssetID,
		}
	}

	lock := domain.UtxoToLock(utxo)
	err = taxi.lockRepository.Lock(ctx, []domain.Lock{lock})

	if err != nil {
		return nil, err
	}

	return utxo, nil
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
	w, err := wallet.NewWallet(taxi.mnemonic, rootPath, taxi.network)
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
			Asset:  taxi.network.AssetID,
			Amount: taxi.fixedFeeAmount,
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
