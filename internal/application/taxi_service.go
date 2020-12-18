package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/shopspring/decimal"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

const (
	receiveDerivationPath = "m/84'/0'/0'/0/0"
	changeDerivationPath  = "m/84'/0'/0'/1/0"
)

// TaxiService provides interface for user UI
type TaxiService interface {
	TopupWithAsset(assetHash string) (*domain.Topup, error)
	ListAssets() []string
}

type taxiService struct {
	lockRepository   domain.LockRepository
	topupRepository  domain.TopupRepository
	priceRepository  domain.PriceRepository
	explorer          explorer.Explorer
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
	ExplorerSvc     explorer.Explorer
}

// NewTaxiService returns a TaxiService interface
func NewTaxiService(args NewTaxiServiceArgs) (TaxiService, error) {
	return newTaxiService(args)
}

func newTaxiService(args NewTaxiServiceArgs) (*taxiService, error) {
	w, err := wallet.NewWallet(args.Mnemonic)
	if err != nil {
		return nil, err
	}

	return &taxiService{
		lockRepository:   args.LockRepository,
		topupRepository:  args.TopupRepository,
		priceRepository:  args.PriceRepository,
		eplorer:          args.ExplorerSvc,
		wallet:           w,
		fixedFeeAmount:   args.FixedFeeAmount,
		basisPointLocker: &sync.Mutex{},
		network:          args.Network,
	}, nil
}

// TopupWithAsset returns a transaction letting to pay a given amount of fees with assetHash token.
func (taxi *taxiService) TopupWithAsset(assetHash string) (*domain.Topup, error) {
	ttlUtxo := 10 * time.Minute
	emptyContext := context.Background()
	totalFees := taxi.fixedFeeAmount

	// check if asset is available
	// price will be used later
	price, assetSupported, err := taxi.priceRepository.GetPrice(emptyContext, assetHash)
	if err != nil {
		return nil, err
	}

	if !assetSupported {
		return nil, ErrAssetNotSupported{AssetHash: assetHash}
	}

	// fetchUtxos + coin select
	utxos, err := taxi.fetchAvailableUtxos()
	if err != nil {
		return nil, err
	}

	utxo, err := selectOneUtxo(utxos, totalFees, taxi.network.AssetID)
	if err != nil {
		return nil, err
	}

	// Lock utxo
	utxoKey := utxo.Key()

	err = taxi.lockRepository.Add(emptyContext, []domain.Lock{utxoKey})
	if err != nil {
		return nil, err
	}

	go func() {
		time.Sleep(ttlUtxo)
		taxi.lockRepository.Remove(emptyContext, utxoKey)
	}()

	expireAt := time.Now().Add(ttlUtxo)

	// request UTXO
	priceToFloat, _ := price.Amount.Float64()
	topup := domain.NewTopup(assetHash, totalFees, priceToFloat, expireAt)
	err = taxi.topupRepository.AddTopup(emptyContext, topup)
	if err != nil {
		return nil, err
	}

	_, err = taxi.topupRepository.SetRequested(emptyContext, topup.ID)
	if err != nil {
		return nil, err
	}

	amountAsDecimal := decimal.NewFromInt(int64(taxi.fixedFeeAmount))
	assetAmount := uint64(price.Amount.Mul(amountAsDecimal).IntPart())

	// create transaction
	taxiPset, err := taxi.createPset(utxo, totalFees, assetHash, assetAmount)
	if err != nil {
		return nil, err
	}

	updater, err := pset.NewUpdater(taxiPset)
	if err != nil {
		return nil, err
	}

	// add prevout to input
	prevout, err := fetchPrevout(taxi.eplorer, utxo)
	if err != nil {
		return nil, err
	}

	if err := updater.AddInWitnessUtxo(prevout, 0); err != nil {
		return nil, err
	}

	receivePayment, err := taxi.getReceivePayment()
	if err != nil {
		return nil, err
	}

	changePayment, err := taxi.getChangePayment()
	if err != nil {
		return nil, err
	}

	// sign the input
	path := receiveDerivationPath
	script := receivePayment.Script

	if bytes.Equal(prevout.Script, changePayment.Script) {
		path = changeDerivationPath
		script = changePayment.Script
	}

	signOpts := wallet.SignInputArgs{
		Updater:       updater,
		UnsignedTx:    taxiPset.UnsignedTx,
		Path:          path,
		InputIndex:    0,
		WitnessUtxo:   prevout,
		WitnessScript: script,
		HashType:      txscript.SigHashSingle + txscript.SigHashAnyOneCanPay,
	}

	if err := taxi.wallet.SignInput(signOpts); err != nil {
		return nil, err
	}

	// check the signatures
	valid, err := taxiPset.ValidateAllSignatures()
	if err != nil {
		return nil, err
	}

	if !valid {
		return nil, ErrInvalidSignature
	}

	partialBase64, err := taxiPset.ToBase64()
	if err != nil {
		return nil, err
	}

	topupCompleted, err := taxi.topupRepository.SetCompleted(emptyContext, topup.ID, partialBase64, assetAmount)
	if err != nil {
		return nil, err
	}

	return topupCompleted, nil
}

// ListAssets return all the available assets of the taxi service
func (taxi *taxiService) ListAssets() []string {
	prices, err := taxi.priceRepository.GetAllPrices(context.Background())
	if err != nil {
		return nil
	}

	supportedAssets := make([]string, 0)

	for _, price := range prices {
		supportedAssets = append(supportedAssets, price.AssetHash)
	}

	return supportedAssets
}

// createPset is an util function used to generate taxi transaction
func (taxi *taxiService) createPset(
	utxo explorer.Utxo,
	totalFees uint64,
	assetHash string,
	assetAmount uint64,
) (*pset.Pset, error) {
	// create taxi tx
	inputs := make([]*transaction.TxInput, 1)
	outputs := make([]*transaction.TxOutput, 2)

	// inputs
	input := transaction.NewTxInput(utxo.Hash(), utxo.Key().Vout)
	inputs[0] = input

	// outputs

	// 1. payout output
	payoutElementsValue, err := elementsutil.SatoshiToElementsValue(uint64(assetAmount))
	if err != nil {
		return nil, err
	}

	taxiPayment, err := taxi.getReceivePayment()
	if err != nil {
		return nil, err
	}

	payoutOutput := transaction.NewTxOutput(
		assetHashBytes(assetHash),
		payoutElementsValue,
		taxiPayment.WitnessScript,
	)

	//	 /!\ payoutOutput MUST be output index 0 /!\
	outputs[0] = payoutOutput

	// 2. fees output
	feesElementsValue, err := elementsutil.SatoshiToElementsValue(totalFees)
	if err != nil {
		return nil, err
	}

	LBTCbytes := assetHashBytes(taxi.network.AssetID)
	feesOutput := transaction.NewTxOutput(
		LBTCbytes,
		feesElementsValue,
		[]byte{},
	)

	outputs[1] = feesOutput

	pset, err := pset.New(inputs, outputs, 2, 0)
	if err != nil {
		return nil, err
	}

	return pset, nil
}

// utils function using to fund the service
func (taxi *taxiService) getAddresses() (string, string, error) {
	rcvPayment, err := taxi.getReceivePayment()
	if err != nil {
		return "", "", err
	}

	rcvAddress, err := rcvPayment.WitnessPubKeyHash()
	if err != nil {
		return "", "", err
	}

	changePayment, err := taxi.getChangePayment()
	if err != nil {
		return "", "", err
	}

	changeAddress, err := changePayment.WitnessPubKeyHash()
	if err != nil {
		return "", "", err
	}

	return rcvAddress, changeAddress, nil
}

func (taxi *taxiService) fetchAvailableUtxos() ([]explorer.Utxo, error) {
	rcvAddress, changeAddress, err := taxi.getAddresses()
	if err != nil {
		return nil, err
	}

	utxosChange, err := taxi.eplorer.GetUnspents(changeAddress)
	if err != nil {
		return nil, err
	}

	utxosReceive, err := taxi.eplorer.GetUnspents(rcvAddress)
	if err != nil {
		return nil, err
	}

	utxos := append(utxosChange, utxosReceive...)

	availableUtxos := make([]explorer.Utxo, 0)

	for _, utxo := range utxos {
		if !taxi.lockRepository.Contains(context.Background(), utxo.Key()) {
			availableUtxos = append(availableUtxos, utxo)
		}
	}

	return availableUtxos, nil
}

func (taxi *taxiService) getReceivePayment() (*payment.Payment, error) {
	return taxi.wallet.DerivePath(receiveDerivationPath, &taxi.network)
}

func (taxi *taxiService) getChangePayment() (*payment.Payment, error) {
	return taxi.wallet.DerivePath(changeDerivationPath, &taxi.network)
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

func assetHashBytes(asset string) []byte {
	buf, _ := hex.DecodeString(asset)
	assetBuf := append([]byte{0x01}, elementsutil.ReverseBytes(buf)...)
	return assetBuf
}

func fetchPrevout(explorer explorer.Explorer, utxo explorer.Utxo) (*transaction.TxOutput, error) {
	hex, err := explorer.FetchTx(utxo.Key().TxID)
	if err != nil {
		return nil, err
	}

	prevTx, err := transaction.NewTxFromHex(hex)
	if err != nil {
		return nil, err
	}

	return prevTx.Outputs[int(utxo.Key().Vout)], nil
}
