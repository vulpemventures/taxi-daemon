package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/google/uuid"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
	"github.com/vulpemventures/liquid-taxi/pkg/fees"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

const (
	receiveDerivationPath = "m/84/0/0/0/0"
	changeDerivationPath  = "m/84/0/0/1/0"
)

type TaxiService interface {
	TopupWithAsset(assetHash string, feeAmount uint64, satPerBytes int) (*domain.Topup, error)
	ListAssets() []string
	getAddresses() (string, string)
}

type taxiService struct {
	utxosLocker     domain.LockRepository
	topupRepository domain.TopupRepository
	eplorer         ExplorerService
	wallet          wallet.Wallet
	changePayment   *payment.Payment
	changeAddress   string
	receivePayment  *payment.Payment
	receiveAddress  string
	network         network.Network
	basisPoint      uint32
}

// NewTaxiService ...
func NewTaxiService(explorerEndpoint string, mnemonic []string, network network.Network, basisPoint uint32) (TaxiService, error) {
	w, err := wallet.NewWallet(mnemonic)
	if err != nil {
		return nil, err
	}

	receivePayment, err := w.DerivePath(receiveDerivationPath, &network)
	if err != nil {
		return nil, err
	}

	receiveAddr, err := receivePayment.WitnessPubKeyHash()
	if err != nil {
		return nil, err
	}

	changePayment, err := w.DerivePath(changeDerivationPath, &network)
	if err != nil {
		return nil, err
	}

	changeAddr, err := changePayment.WitnessPubKeyHash()
	if err != nil {
		return nil, err
	}

	return &taxiService{
		utxosLocker:     inmemory.NewInmemoryLockRepository(),
		topupRepository: inmemory.NewInmemoryTopupRepository(),
		eplorer:         NewEsplorerService(explorerEndpoint),
		wallet:          w,
		changePayment:   changePayment,
		changeAddress:   changeAddr,
		receivePayment:  receivePayment,
		receiveAddress:  receiveAddr,
		basisPoint:      basisPoint,
		network:         network,
	}, nil
}

func (taxi *taxiService) TopupWithAsset(assetHash string, feeAmount uint64, satPerBytes int) (*domain.Topup, error) {
	ttlUtxo := 10 * time.Minute

	// TO DO : check if asset is available

	emptyContext := context.Background()

	// fetchUtxos + coin select
	utxos, err := taxi.fetchAvailableUtxos()
	if err != nil {
		return nil, err
	}

	selectedUtxos, changeAmount, err := coinSelect(utxos, feeAmount, taxi.network.AssetID)
	if err != nil {
		return nil, err
	}

	// estimate fee increase
	feeIncrease := fees.CalculateFeeIncrease(len(selectedUtxos), 3, satPerBytes)
	totalFees := feeAmount + feeIncrease

	if changeAmount >= feeIncrease {
		changeAmount -= feeIncrease
	} else {
		selectedUtxos, changeAmount, err = coinSelect(utxos, totalFees, taxi.network.AssetID)
		if err != nil {
			return nil, err
		}
	}

	expireAt := time.Now().Add(ttlUtxo)
	// Lock utxos
	for _, utxo := range selectedUtxos {
		utxoKey := utxo.Key()

		if taxi.utxosLocker.Contains(emptyContext, utxoKey) {
			return nil, ErrUtxoNotAvailable
		}

		err := taxi.utxosLocker.Add(emptyContext, []domain.Lock{utxoKey})
		if err != nil {
			return nil, err
		}

		go func() {
			time.Sleep(ttlUtxo)
			taxi.utxosLocker.Remove(emptyContext, utxoKey)
		}()
	}

	// TO DO: fetch the prices
	assetAmount := 10 * totalFees
	assetAmount += uint64(float64(assetAmount) * float64(taxi.basisPoint) / 100)

	// create transaction
	taxiPset, err := taxi.createPset(selectedUtxos, totalFees, changeAmount, assetHash, assetAmount)
	if err != nil {
		return nil, err
	}

	updater, err := pset.NewUpdater(taxiPset)
	if err != nil {
		return nil, err
	}

	prevouts := make([]*transaction.TxOutput, 0)
	// add prevouts to pset inputs
	for inputIndex, utxo := range selectedUtxos {
		prevout, err := taxi.eplorer.FetchPrevout(utxo)
		if err != nil {
			return nil, err
		}

		prevouts = append(prevouts, prevout)

		if err := updater.AddInWitnessUtxo(prevout, inputIndex); err != nil {
			return nil, err
		}
	}

	// sign with wallet
	for index, witnessUtxo := range prevouts {
		path := receiveDerivationPath
		script := taxi.receivePayment.Script

		if bytes.Equal(witnessUtxo.Script, taxi.changePayment.Script) {
			path = changeDerivationPath
			script = taxi.changePayment.Script
		}

		signOpts := wallet.SignInputArgs{
			Updater:       updater,
			UnsignedTx:    taxiPset.UnsignedTx,
			Path:          path,
			InputIndex:    index,
			WitnessUtxo:   witnessUtxo,
			WitnessScript: script,
			HashType:      txscript.SigHashAll + txscript.SigHashAnyOneCanPay,
		}

		if err := taxi.wallet.SignInput(signOpts); err != nil {
			return nil, err
		}
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

	topup := domain.Topup{
		Id:          uuid.New().String(),
		Partial:     partialBase64,
		AssetHash:   assetHash,
		AssetPrice:  10,
		AssetAmount: assetAmount,
		FeeAmount:   totalFees,
		BasisPoint:  int32(taxi.basisPoint),
		ExpireAt:    expireAt,
	}

	err = taxi.topupRepository.AddTopup(emptyContext, topup)
	if err != nil {
		return nil, err
	}

	err = taxi.topupRepository.SetRequested(emptyContext, topup.Id)
	if err != nil {
		return nil, err
	}

	return &topup, nil
}

func (taxi *taxiService) ListAssets() []string {
	panic("not implemented")
}

func (taxi *taxiService) createPset(
	selectedUtxos []Utxo,
	totalFees uint64,
	changeAmount uint64,
	assetHash string,
	assetAmount uint64,
) (*pset.Pset, error) {
	// create taxi tx
	inputs := make([]*transaction.TxInput, 0)
	outputs := make([]*transaction.TxOutput, 0)

	// inputs
	for _, utxo := range selectedUtxos {
		input := transaction.NewTxInput(utxo.Hash(), utxo.Vout)
		inputs = append(inputs, input)
	}

	// outputs

	// 1. fees output
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

	outputs = append(outputs, feesOutput)

	// 2. payout output
	// TO DO : fetch the prices
	payoutElementsValue, err := elementsutil.SatoshiToElementsValue(uint64(assetAmount))
	if err != nil {
		return nil, err
	}

	payoutOutput := transaction.NewTxOutput(
		assetHashBytes(assetHash),
		payoutElementsValue,
		taxi.receivePayment.WitnessScript,
	)

	outputs = append(outputs, payoutOutput)

	// 3. The change output
	if changeAmount > 0 {
		changeElementsValue, err := elementsutil.SatoshiToElementsValue(changeAmount)
		if err != nil {
			return nil, err
		}

		changeOutput := transaction.NewTxOutput(
			LBTCbytes,
			changeElementsValue,
			taxi.changePayment.WitnessScript,
		)

		outputs = append(outputs, changeOutput)
	}

	pset, err := pset.New(inputs, outputs, 2, 0)
	if err != nil {
		return nil, err
	}

	return pset, nil
}

func (taxi *taxiService) getAddresses() (string, string) {
	return taxi.receiveAddress, taxi.changeAddress
}

func (taxi *taxiService) fetchAvailableUtxos() ([]Utxo, error) {
	utxosChange, err := taxi.eplorer.GetUnspents(taxi.changeAddress)
	if err != nil {
		return nil, err
	}

	utxosReceive, err := taxi.eplorer.GetUnspents(taxi.receiveAddress)
	if err != nil {
		return nil, err
	}

	utxos := append(utxosChange, utxosReceive...)

	availableUtxos := make([]Utxo, 0)

	for _, utxo := range utxos {
		if !taxi.utxosLocker.Contains(context.Background(), utxo.Key()) {
			availableUtxos = append(availableUtxos, utxo)
		}
	}

	return availableUtxos, nil
}

func coinSelect(utxos []Utxo, targetAmount uint64, targetAsset string) ([]Utxo, uint64, error) {
	var totalAmount uint64 = 0
	selectedUtxos := make([]Utxo, 0)

	for _, utxo := range utxos {
		if !utxo.IsConfidential() && utxo.Asset == targetAsset {
			totalAmount += utxo.Value
			selectedUtxos = append(selectedUtxos, utxo)
			if totalAmount >= targetAmount {
				break
			}
		}
	}

	if totalAmount < targetAmount {
		return nil, 0, errors.New(
			"error on target amount: total utxo amount does not cover target amount",
		)
	}

	changeAmount := totalAmount - targetAmount

	return selectedUtxos, changeAmount, nil
}

func assetHashBytes(asset string) []byte {
	buf, _ := hex.DecodeString(asset)
	assetBuf := append([]byte{0x01}, elementsutil.ReverseBytes(buf)...)
	return assetBuf
}
