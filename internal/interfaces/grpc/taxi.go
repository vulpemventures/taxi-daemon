package grpc

import (
	"context"

	"github.com/vulpemventures/liquid-taxi/internal/application"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
)

type taxiServer struct {
	pb.UnimplementedTaxiServer
	taxiSvc application.TaxiService
}

// NewTaxiHandler returns a type implementing the pb.TaxiServer interface
func NewTaxiHandler(taxiSvc application.TaxiService) pb.TaxiServer {
	return &taxiServer{
		taxiSvc: taxiSvc,
	}
}

func (t *taxiServer) ListAssets(ctx context.Context, _ *pb.ListAssetsRequest) (*pb.ListAssetsReply, error) {
	assets := t.taxiSvc.ListAssets(ctx)
	return &pb.ListAssetsReply{
		AssetHash: assets,
	}, nil
}

func (t *taxiServer) TopupWithAsset(ctx context.Context, req *pb.TopupWithAssetRequest) (*pb.TopupWithAssetReply, error) {
	topup, err := t.taxiSvc.TopupWithAsset(ctx, req.GetAssetHash())
	if err != nil {
		return nil, err
	}

	return &pb.TopupWithAssetReply{
		Topup:  topup.ToTopupMessage(),
		Expiry: uint64(topup.ExpireAt.Unix()),
	}, nil
}
