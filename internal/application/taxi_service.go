package application

import (
	"context"
	"sync"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/shopspring/decimal"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
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
	ListAssets(ctx context.Context) []string
}

type taxiService struct {
	lockRepository   domain.LockRepository
	topupRepository  domain.TopupRepository
	priceRepository  domain.PriceRepository
	explorer         explorer.Service
	wallet           wallet.Wallet
	network          network.Network
	fixedFeeAmount   uint64
	basisPointLocker sync.Locker
}

// NewTaxiServiceArgs is a wrapper type for NewTaxiService factory function
type NewTaxiServiceArgs struct {
	Mnemonic        []string
	Network         network.Network
	FixedFeeAmount  uint64
	TopupRepository domain.TopupRepository
	LockRepository  domain.LockRepository
	PriceRepository domain.PriceRepository
	ExplorerSvc     explorer.Service
}

// NewTaxiService returns a TaxiService interface
func NewTaxiService(args NewTaxiServiceArgs) (TaxiService, error) {
	return newTaxiService(args)
}

func newTaxiService(args NewTaxiServiceArgs) (*taxiService, error) {
	w, err := wallet.NewWallet(args.Mnemonic, args.Network)
	if err != nil {
		return nil, err
	}

	return &taxiService{
		lockRepository:   args.LockRepository,
		topupRepository:  args.TopupRepository,
		priceRepository:  args.PriceRepository,
		explorer:         args.ExplorerSvc,
		wallet:           w,
		fixedFeeAmount:   args.FixedFeeAmount,
		basisPointLocker: &sync.Mutex{},
		network:          args.Network,
	}, nil
}

// TopupWithAsset returns a transaction letting to pay a given amount of fees with assetHash token.
func (taxi *taxiService) TopupWithAsset(ctx context.Context, assetHash string) (*domain.Topup, error) {
	ttlUtxo := 10 * time.Minute

	// check if asset is available
	// price will be used later
	price, assetSupported, err := taxi.priceRepository.GetPrice(ctx, assetHash)
	if err != nil {
		return nil, err
	}

	if !assetSupported {
		return nil, ErrAssetNotSupported{AssetHash: assetHash}
	}

	// retrieve only the utxos that are not locked by other topups
	utxos, err := taxi.fetchAvailableUtxos(ctx)
	if err != nil {
		return nil, err
	}

	utxo, err := selectOneUtxo(utxos, taxi.fixedFeeAmount, taxi.network.AssetID)
	if err != nil {
		return nil, err
	}

	expireAt := time.Now().Add(ttlUtxo)

	topup := domain.NewTopup(assetHash, taxi.fixedFeeAmount, price.Amount, expireAt)
	err = taxi.topupRepository.AddTopup(ctx, topup)
	if err != nil {
		return nil, err
	}

	amountAsDecimal := decimal.NewFromInt(int64(taxi.fixedFeeAmount))
	assetAmount := uint64(price.Amount.Mul(amountAsDecimal).IntPart())

	payment, err := taxi.getPaymentForAsset()
	if err != nil {
		return nil, err
	}

	outputsOpts := []wallet.PsetOutputOpts{
		wallet.PsetOutputOpts{
			WitnessScript: payment.WitnessScript,
			Asset:         assetHash,
			Amount:        assetAmount,
		},
		wallet.PsetOutputOpts{
			WitnessScript: []byte{},
			Asset:         taxi.network.AssetID,
			Amount:        taxi.fixedFeeAmount,
		},
	}

	// create transaction
	taxiPset, err := taxi.wallet.CreatePSET([]explorer.Utxo{utxo}, outputsOpts)
	if err != nil {
		return nil, err
	}

	signOpts := wallet.SignInputArgs{
		PsetBase64: taxiPset,
		Path:       derivationPathForLBTC,
		InputIndex: 0,
		HashType:   txscript.SigHashSingle | txscript.SigHashAnyOneCanPay,
	}

	signedPsetBase64, err := taxi.wallet.SignInput(signOpts)
	if err != nil {
		return nil, err
	}

	topupCompleted, err := taxi.topupRepository.SetCompleted(ctx, topup.ID, signedPsetBase64, assetAmount)
	if err != nil {
		return nil, err
	}

	// Lock the utxo used for this topup to avoid other ones to use it
	utxoKey := domain.UtxoToLock(utxo)
	err = taxi.lockRepository.Add(ctx, []domain.Lock{utxoKey})
	if err != nil {
		return nil, err
	}

	go func() {
		time.Sleep(ttlUtxo)
		taxi.lockRepository.Remove(ctx, utxoKey)
	}()

	return topupCompleted, nil
}

// ListAssets return all the available assets of the taxi service
func (taxi *taxiService) ListAssets(ctx context.Context) []string {
	prices, err := taxi.priceRepository.GetAllPrices(ctx)
	if err != nil {
		return nil
	}

	supportedAssets := make([]string, 0)

	for _, price := range prices {
		supportedAssets = append(supportedAssets, price.AssetHash)
	}

	return supportedAssets
}

// utils function using to fund the service
func (taxi *taxiService) getFundAddress() (string, error) {
	paymentForLBTC, err := taxi.getPaymentForLBTC()
	if err != nil {
		return "", err
	}

	lbtcAddress, err := paymentForLBTC.WitnessPubKeyHash()
	if err != nil {
		return "", err
	}

	// TODO now the taxi service has only two addresses. In the future taxi will generate a set of addresses for each topup
	return lbtcAddress, nil
}

func (taxi *taxiService) fetchAvailableUtxos(ctx context.Context) ([]explorer.Utxo, error) {
	address, err := taxi.getFundAddress()
	if err != nil {
		return nil, err
	}

	utxos, err := taxi.explorer.GetUnspents(address, [][]byte{})
	if err != nil {
		return nil, err
	}

	availableUtxos := make([]explorer.Utxo, 0)

	for _, utxo := range utxos {
		utxoKey := domain.UtxoToLock(utxo)

		if !taxi.lockRepository.Contains(ctx, utxoKey) {
			availableUtxos = append(availableUtxos, utxo)
		}
	}

	return availableUtxos, nil
}

func (taxi *taxiService) getPaymentForAsset() (*payment.Payment, error) {
	return taxi.wallet.DerivePath(derivationPathForAsset, &taxi.network)
}

func (taxi *taxiService) getPaymentForLBTC() (*payment.Payment, error) {
	return taxi.wallet.DerivePath(derivationPathForLBTC, &taxi.network)
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
