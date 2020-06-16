package taxiserver

import (
	"context"

	"github.com/twitchtv/twirp"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/pkg/price"
	pb "github.com/vulpemventures/liquid-taxi/rpc/taxi"
)

var assets = map[string]map[string]bool{
	"liquid": {
		"ce091c998b83c78bb71a632313ba3760f1763d9cfcffae02258ffa9865a37bd2": true,
	},
	"regtest": {
		"ce091c998b83c78bb71a632313ba3760f1763d9cfcffae02258ffa9865a37bd2": true,
	},
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
	if req.Unsigned == nil {
		return nil, twirp.InvalidArgumentError("unsigned", "is missing")
	}
	if req.AssetHash == "" {
		return nil, twirp.InvalidArgumentError("asset_hash", "is missing")
	}

	if _, ok := assets[config.GetString(config.NetworkKey)][req.AssetHash]; !ok {
		return nil, twirp.InvalidArgumentError("asset_hash", "not accepted as payment")
	}

	// fake fee calculation
	// TODO Add actual calculation with fees package
	var satPerByte = float32(0.1)
	if req.Unsigned.SatPerByte > satPerByte {
		satPerByte = req.Unsigned.SatPerByte
	}
	size := uint64(500)
	fees := uint64(float32(size) * satPerByte)
	rate := float32(0.1)
	spread := uint64(float32(fees) * rate)
	total := fees + spread

	totalFeesInAsset, err := price.BitcoinToUsdt(total)
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}

	return &pb.GetAssetEstimateReply{
		AssetAmount: totalFeesInAsset,
		AssetHash:   req.AssetHash,
		Breakdown: &pb.Breakdown{
			Size:       size,
			SatPerByte: satPerByte,
			Fee:        fees,
			Rate:       rate,
			Spread:     spread,
			Total:      total,
		},
	}, nil
}

// TopupWithAsset implements TopupWithAsset rpc
func (s *Server) TopupWithAsset(ctx context.Context, req *pb.TopupWithAssetRequest) (res *pb.TopupWithAssetReply, err error) {
	return &pb.TopupWithAssetReply{}, nil
}
