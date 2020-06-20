package taxiserver

import (
	"context"
	"time"

	"github.com/tiero/ocean/pkg/coinselect"
	"github.com/tiero/ocean/pkg/explorer/blockstream"
	"github.com/tiero/ocean/pkg/keypair"
	"github.com/tiero/ocean/pkg/partial"
	"github.com/twitchtv/twirp"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/pkg/fees"
	"github.com/vulpemventures/liquid-taxi/pkg/price"

	pb "github.com/vulpemventures/taxi-protobuf/rpc/taxi"
)

//ASSETS ...
var ASSETS = map[string]map[string]bool{
	"liquid": {
		"ce091c998b83c78bb71a632313ba3760f1763d9cfcffae02258ffa9865a37bd2": true,
	},
	"regtest": {
		"": true,
	},
}

//NETWORK ...
var NETWORK = map[string]network.Network{
	"liquid":  network.Liquid,
	"regtest": network.Regtest,
}

// Server implements the Taxi service
type Server struct{}

// Topup implements Topup rpc
func (s *Server) Topup(ctx context.Context, req *pb.TopupRequest) (res *pb.TopupReply, err error) {
	return &pb.TopupReply{}, nil
}

// GetTopup implements GetTopup rpc
func (s *Server) GetTopup(ctx context.Context, req *pb.GetTopupRequest) (res *pb.GetTopupReply, err error) {
	return &pb.GetTopupReply{}, nil
}

// TopupWithKey implements TopupWithKey rpc
func (s *Server) TopupWithKey(ctx context.Context, req *pb.TopupWithKeyRequest) (res *pb.TopupWithKeyReply, err error) {
	return &pb.TopupWithKeyReply{}, nil
}

// GetAssetEstimate implements GetAssetEstimate rpc
func (s *Server) GetAssetEstimate(ctx context.Context, req *pb.GetAssetEstimateRequest) (res *pb.GetAssetEstimateReply, err error) {
	if req.Unsigned == nil || req.Unsigned.GetPset() == "" {
		return nil, twirp.InvalidArgumentError("unsigned pset", "is missing")
	}
	if req.AssetHash == "" {
		return nil, twirp.InvalidArgumentError("asset_hash", "is missing")
	}

	if !acceptedAsset(req.GetAssetHash()) {
		return nil, twirp.InvalidArgumentError("asset_hash", "not accepted as payment")
	}

	decoded, err := pset.NewPsetFromBase64(req.Unsigned.GetPset())
	if err != nil {
		return nil, err
	}

	breakdown, err := fees.CalculateWitness(1+len(decoded.Inputs), 2+len(decoded.Outputs), req.Unsigned.GetSatPerByte(), 0.1)
	if err != nil {
		return nil, twirp.InvalidArgumentError("pset", "is not valid")
	}
	prices, err := price.GetPriceOfPair("XBTUSDT")
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}
	usdtPrice := prices[0]
	totalFeesInAsset := price.BitcoinToUsdt(breakdown.Total, usdtPrice)

	return &pb.GetAssetEstimateReply{
		Breakdown: &pb.Breakdown{
			Size:       breakdown.Size,
			Fee:        breakdown.Fee,
			Spread:     breakdown.Spread,
			SatPerByte: breakdown.SatPerByte,
			Rate:       breakdown.Rate,
			Total:      breakdown.Total,
		},
		AssetAmount: totalFeesInAsset,
		AssetHash:   req.AssetHash,
	}, nil
}

// TopupWithAsset implements TopupWithAsset rpc
func (s *Server) TopupWithAsset(ctx context.Context, req *pb.TopupWithAssetRequest) (res *pb.TopupWithAssetReply, err error) {
	if req.Unsigned == nil || req.Unsigned.GetPset() == "" {
		return nil, twirp.InvalidArgumentError("unsigned pset", "is missing")
	}
	if req.AssetHash == "" {
		return nil, twirp.InvalidArgumentError("asset_hash", "is missing")
	}
	if !acceptedAsset(req.GetAssetHash()) {
		return nil, twirp.InvalidArgumentError("asset_hash", "not accepted as payment")
	}

	currentNetwork := NETWORK[config.GetString(config.NetworkKey)]

	decoded, err := pset.NewPsetFromBase64(req.Unsigned.GetPset())
	if err != nil {
		return nil, err
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
		return nil, twirp.InternalError(err.Error())
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
		return nil, twirp.InternalError(err.Error())
	}

	BitcoinAsset := NETWORK[config.GetString(config.NetworkKey)].AssetID
	selectedUtxos, change, err := coinselect.CoinSelect(taxiUtxos, breakdown.Fee, BitcoinAsset)
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}

	psetWithFees := &partial.Partial{Data: decoded, Network: &currentNetwork}
	for _, utxo := range selectedUtxos {
		psetWithFees.AddInput(utxo.Hash(), utxo.Index(), &partial.WitnessUtxo{Asset: BitcoinAsset, Value: utxo.Value(), Script: taxiPayment.Script}, nil)
	}
	// payout to taxi expressed in Asset
	psetWithFees.AddOutput(req.AssetHash, totalFeesInAsset, taxiPayment.Script, false)
	// Taxi adds Fee LBTC output and his change
	psetWithFees.AddOutput(BitcoinAsset, change, taxiPayment.Script, false)
	psetWithFees.AddOutput(BitcoinAsset, breakdown.Fee, []byte{}, false)

	for i := givenInputsLen; i < len(psetWithFees.Data.Inputs); i++ {
		err := psetWithFees.SignWithPrivateKey(i, keyPair)
		if err != nil {
			return nil, twirp.InternalError(err.Error())
		}
	}

	base64, err := psetWithFees.Data.ToBase64()
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}

	return &pb.TopupWithAssetReply{
		Order: &pb.Order{
			//TODO Generate proper ORDER ID to persist the topup
			OrderId:     "random1234",
			Status:      pb.Order_UNPAID,
			IsCompleted: false,
			Unsigned:    req.Unsigned,
			Breakdown: &pb.Breakdown{
				Size:       breakdown.Size,
				Fee:        breakdown.Fee,
				Spread:     breakdown.Spread,
				SatPerByte: breakdown.SatPerByte,
				Rate:       breakdown.Rate,
				Total:      breakdown.Total,
			},
			Partial: &pb.PartialWithFees{
				Pset: base64,
			},
			CreatedAt: time.Now().Unix(),
			Method:    pb.Order_ASSET,
		},
		AssetAmount: totalFeesInAsset,
		AssetHash:   req.AssetHash,
	}, nil
}

func acceptedAsset(asset string) bool {
	net := config.GetString(config.NetworkKey)

	if net != "liquid" {
		return true
	}

	if _, ok := ASSETS[net][asset]; !ok {
		return false
	}

	return true
}
