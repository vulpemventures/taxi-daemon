package taxiserver

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/twitchtv/twirp"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/explorer"
	"github.com/vulpemventures/liquid-taxi/internal/storage"

	pb "github.com/vulpemventures/taxi-protobuf/rpc/taxi"
)

func getTaxiPrivateKey() (*btcec.PrivateKey, error) {
	bytes, err := hex.DecodeString(config.GetString(config.PrivateKeyKey))
	if err != nil {
		return nil, err
	}
	privateKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), bytes)
	return privateKey, nil
}

// TopupWithAsset implements TopupWithAsset rpc
func (s *Server) TopupWithAsset(ctx context.Context, req *pb.TopupWithAssetRequest) (res *pb.TopupWithAssetReply, err error) {
	orderID := req.GetOrderId()
	if orderID == "" {
		return nil, twirp.RequiredArgumentError("order_id")
	}

	if req.Unsigned == nil || req.Unsigned.GetPset() == "" {
		return nil, twirp.RequiredArgumentError("unsigned.pset")
	}
	if req.AssetHash == "" {
		return nil, twirp.RequiredArgumentError("asset_hash")
	}

	if !notAcceptedAsset(req.GetAssetHash()) {
		return nil, twirp.InvalidArgumentError("asset_hash", "not accepted as payment")
	}

	// Mongo DB connection
	store, err := storage.NewStorage(config.GetString(config.MongoEndpoint))
	if err != nil {
		return nil, twirp.InternalErrorWith(fmt.Errorf("Storage: %w", err))
	}

	// Get back the whole order to show to the user
	order, notFound, err := store.GetOrderByID(orderID)
	if !notFound && err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	if notFound {
		return nil, twirp.InternalErrorWith(err)
	}

	currentNetwork := NETWORK[config.GetString(config.NetworkKey)]

	decoded, err := pset.NewPsetFromBase64(req.Unsigned.GetPset())
	if err != nil {
		return nil, twirp.InvalidArgumentError("pset", "not valid")
	}
	givenInputsLen := len(decoded.Inputs)
	totalFeesInAsset := order.AssetAmount

	// Taxi Key Pair
	privateKey, err := getTaxiPrivateKey()
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}
	taxiPayment := payment.FromPublicKey(privateKey.PubKey(), &currentNetwork, nil)
	taxiAddress, err := taxiPayment.WitnessPubKeyHash()
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}

	// Retrieve Utxos for Taxi address
	e := explorer.NewEsplorerService(config.GetString(config.ExplorerEndpointKey))

	taxiUtxos, err := e.GetUnspents(taxiAddress)
	if err != nil {
		return nil, twirp.InternalErrorWith(fmt.Errorf("Explorer: %w", err))
	}

	BitcoinAsset := NETWORK[config.GetString(config.NetworkKey)].AssetID
	BitcoinFee := order.Breakdown.Fee
	selectedUtxos, change, err := coinSelect(taxiUtxos, BitcoinFee, BitcoinAsset)
	if err != nil {
		return nil, twirp.InternalErrorWith(fmt.Errorf("Coin Select: %w", err))
	}

	psetUpdater, err := pset.NewUpdater(decoded)
	if err != nil {
		return nil, err
	}

	for index, utxo := range selectedUtxos {
		isLocked, err := store.CheckLock(utxo.TxId, int(utxo.Vout))
		if err != nil {
			return nil, twirp.InternalErrorWith(fmt.Errorf("Storage: %w", err))
		}

		if !isLocked {
			input := transaction.NewTxInput(utxo.Hash(), utxo.Vout)
			prevout, err := utxo.FetchPrevout(e)
			if err != nil {
				return nil, err
			}

			psetUpdater.AddInput(input)
			err = psetUpdater.AddInWitnessUtxo(prevout, givenInputsLen+index)
			if err != nil {
				return nil, err
			}

			// Lock the utxo
			if err := store.LockUnspent(utxo.TxId, int(utxo.Vout)); err != nil {
				return nil, twirp.InternalErrorWith(fmt.Errorf("Storage: %w", err))
			}
		}
	}
	// this means we don't have any unlocked utxo to spend.
	if len(decoded.Inputs) == givenInputsLen {
		return nil, twirp.InternalError("All utxos are locked. Try in again in 5 minutes")
	}

	// create elements values
	feesInAssetValue, err := elementsutil.SatoshiToElementsValue(totalFeesInAsset)
	if err != nil {
		return nil, err
	}

	changeValue, err := elementsutil.SatoshiToElementsValue(change)
	if err != nil {
		return nil, err
	}

	bitcoinFeeValue, err := elementsutil.SatoshiToElementsValue(BitcoinFee)
	if err != nil {
		return nil, err
	}

	// create the outputs
	payoutToTaxiOutput := transaction.NewTxOutput(assetHash(req.AssetHash), feesInAssetValue, taxiPayment.WitnessScript)

	changeOutput := transaction.NewTxOutput(assetHash(BitcoinAsset), changeValue, taxiPayment.WitnessScript)

	FeeOutput := transaction.NewTxOutput(assetHash(BitcoinAsset), bitcoinFeeValue, []byte{})

	// add the outputs to pset
	outputs := []*transaction.TxOutput{payoutToTaxiOutput, changeOutput, FeeOutput}
	for _, output := range outputs {
		psetUpdater.AddOutput(output)
	}

	// NOTICE: here we start the loop from givenInputsLenght so we sign ony the taxi inputs.
	for i := givenInputsLen; i < len(decoded.Inputs); i++ {
		if err := psetUpdater.AddInSighashType(txscript.SigHashAll, i); err != nil {
			return nil, err
		}

		sigHash := decoded.UnsignedTx.HashForWitnessV0(
			i,
			taxiPayment.Script,
			decoded.Inputs[i].WitnessUtxo.Value,
			txscript.SigHashAll,
		)

		sig, err := privateKey.Sign(sigHash[:])
		if err != nil {
			return nil, err
		}

		sigWithHashType := append(sig.Serialize(), byte(txscript.SigHashAll))

		_, err = psetUpdater.Sign(i, sigWithHashType, privateKey.PubKey().SerializeCompressed(), nil, nil)
		if err != nil {
			return nil, err
		}
	}

	base64, err := decoded.ToBase64()
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	// Persist order for history purpose and lock utxos
	err = store.UpdateOrderWithTransactions(orderID, req.Unsigned.GetPset(), base64)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	// Get back the whole order to show to the user
	order, notFound, err = store.GetOrderByID(orderID)
	if !notFound && err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	if notFound {
		return nil, twirp.InternalErrorWith(err)
	}

	if err := store.Close(); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	res = &pb.TopupWithAssetReply{
		Order:       order.ToProtobuf(orderID),
		AssetAmount: totalFeesInAsset,
		AssetHash:   req.AssetHash,
	}

	return res, nil
}

func coinSelect(utxos []explorer.Utxo, targetAmount uint64, targetAsset string) ([]explorer.Utxo, uint64, error) {
	var totalAmount uint64 = 0
	selectedUtxos := make([]explorer.Utxo, 0)

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

func assetHash(asset string) []byte {
	buf, _ := hex.DecodeString(asset)
	assetBuf := append([]byte{0x01}, elementsutil.ReverseBytes(buf)...)
	return assetBuf
}
