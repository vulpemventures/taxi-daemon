package taxiserver

import (
	"context"

	"github.com/twitchtv/twirp"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/liquid-taxi/pkg/fees"
	"github.com/vulpemventures/liquid-taxi/pkg/price"

	pb "github.com/vulpemventures/taxi-protobuf/rpc/taxi"
)

// GetAssetEstimate implements GetAssetEstimate rpc
func (s *Server) GetAssetEstimate(ctx context.Context, req *pb.GetAssetEstimateRequest) (res *pb.GetAssetEstimateReply, err error) {
	if req.Unsigned == nil || req.Unsigned.GetPset() == "" {
		return nil, twirp.InvalidArgumentError("unsigned pset", "is missing")
	}
	if req.AssetHash == "" {
		return nil, twirp.InvalidArgumentError("asset_hash", "is missing")
	}

	if !notAcceptedAsset(req.GetAssetHash()) {
		return nil, twirp.InvalidArgumentError("asset_hash", "not accepted as payment")
	}

	decoded, err := pset.NewPsetFromBase64(req.Unsigned.GetPset())
	if err != nil {
		return nil, twirp.InvalidArgumentError("pset", "not valid")
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

	res = &pb.GetAssetEstimateReply{
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
	}

	return res, nil
}
