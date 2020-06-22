package taxiserver

import (
	"context"
	"fmt"

	"github.com/tiero/ocean/pkg/coinselect"
	"github.com/tiero/ocean/pkg/explorer/blockstream"
	"github.com/tiero/ocean/pkg/keypair"
	"github.com/tiero/ocean/pkg/partial"
	"github.com/twitchtv/twirp"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/storage"
	"github.com/vulpemventures/liquid-taxi/pkg/fees"
	"github.com/vulpemventures/liquid-taxi/pkg/price"

	pb "github.com/vulpemventures/taxi-protobuf/rpc/taxi"
)

// TopupWithAsset implements TopupWithAsset rpc
func (s *Server) TopupWithAsset(ctx context.Context, req *pb.TopupWithAssetRequest) (res *pb.TopupWithAssetReply, err error) {
	if req.Unsigned == nil || req.Unsigned.GetPset() == "" {
		return nil, twirp.RequiredArgumentError("unsigned.pset")
	}
	if req.AssetHash == "" {
		return nil, twirp.RequiredArgumentError("asset_hash")
	}

	if !notAcceptedAsset(req.GetAssetHash()) {
		return nil, twirp.InvalidArgumentError("asset_hash", "not accepted as payment")
	}

	currentNetwork := NETWORK[config.GetString(config.NetworkKey)]

	decoded, err := pset.NewPsetFromBase64(req.Unsigned.GetPset())
	if err != nil {
		return nil, twirp.InvalidArgumentError("pset", "not valid")
	}

	// Breakdown
	givenInputsLen := len(decoded.Inputs)
	givenOutputsLen := len(decoded.Outputs)
	breakdown, err := fees.CalculateWitness(1+givenInputsLen, 2+givenOutputsLen, req.Unsigned.GetSatPerByte(), 0.1)
	if err != nil {
		return nil, twirp.InvalidArgumentError("pset", "is not valid")
	}
	// Exchange Prices
	prices, err := price.GetPriceOfPair("XBTUSDT")
	if err != nil {
		return nil, twirp.InternalError(fmt.Errorf("Price: %w", err).Error())
	}
	usdtPrice := prices[0]
	totalFeesInAsset := price.BitcoinToUsdt(breakdown.Total, usdtPrice)

	// Taxi Key Pair
	keyPair, err := keypair.FromPrivateKey(config.GetString(config.PrivateKeyKey))
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}
	taxiPayment := payment.FromPublicKey(keyPair.PublicKey, &currentNetwork, nil)
	taxiAddress, err := taxiPayment.WitnessPubKeyHash()
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}

	// Retrieve Utxos for Taxi address
	e := blockstream.NewExplorer(config.GetString(config.ExplorerEndpointKey))
	taxiUtxos, err := e.GetUnspents(taxiAddress)
	if err != nil {
		return nil, twirp.InternalError(fmt.Errorf("Explorer: %w", err).Error())
	}

	BitcoinAsset := NETWORK[config.GetString(config.NetworkKey)].AssetID
	selectedUtxos, change, err := coinselect.CoinSelect(taxiUtxos, breakdown.Fee, BitcoinAsset)
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}

	store, err := storage.NewStorage(config.GetString(config.MongoEndpoint))
	if err != nil {
		return nil, twirp.InternalError(fmt.Errorf("Storage: %w", err).Error())
	}

	//var locked []*storage.Unspent
	psetWithFees := &partial.Partial{Data: decoded, Network: &currentNetwork}
	for _, utxo := range selectedUtxos {
		txid := utxo.Hash()
		vout := int(utxo.Index())
		if isLocked, _ := store.CheckLock(txid, vout); !isLocked {
			psetWithFees.AddInput(utxo.Hash(), utxo.Index(), &partial.WitnessUtxo{Asset: BitcoinAsset, Value: utxo.Value(), Script: taxiPayment.Script}, nil)
			// Lock the utxo
			if err := store.LockUnspent(txid, vout); err != nil {
				return nil, twirp.InternalErrorWith(err)
			}
			//locked = append(locked, &storage.Unspent{TxID: txid, Vout: vout})
		}
	}
	// this means we don't have any unlocked utxo to spend.
	if len(psetWithFees.Data.Inputs) == givenInputsLen {
		return nil, twirp.InternalError("All utxos are locked. Try in again in 5 minutes")
	}
	// payout to taxi expressed in Asset
	psetWithFees.AddOutput(req.AssetHash, totalFeesInAsset, taxiPayment.Script, false)
	// Taxi adds Fee LBTC output and his change
	psetWithFees.AddOutput(BitcoinAsset, change, taxiPayment.Script, false)
	psetWithFees.AddOutput(BitcoinAsset, breakdown.Fee, []byte{}, false)

	// NOTICE: here we start the loop from givenInputsLenght so we sign ony the taxi inputs.
	for i := givenInputsLen; i < len(psetWithFees.Data.Inputs); i++ {
		err := psetWithFees.SignWithPrivateKey(i, keyPair)
		if err != nil {
			return nil, twirp.InternalErrorWith(err)
		}
	}

	base64, err := psetWithFees.Data.ToBase64()
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	// Persist order for history purpose and lock utxos
	orderID, err := store.InsertOrder(int(pb.Order_ASSET), req.Unsigned.GetPset(), base64, breakdown)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	// Get back the whole order to show to the user
	order, notFound, err := store.GetOrderByID(orderID)
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
