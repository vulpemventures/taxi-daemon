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

var _ pb.TaxiServer = (*taxiServer)(nil)

func (t *taxiServer) ListAssets(ctx context.Context, req *pb.ListAssetsRequest) (*pb.ListAssetsReply, error) {
	panic("not implemented")
}

func (t *taxiServer) TopupWithAsset(ctx context.Context, req *pb.TopupWithAssetRequest) (*pb.TopupWithAssetReply, error) {
	panic("not implemented")
}
