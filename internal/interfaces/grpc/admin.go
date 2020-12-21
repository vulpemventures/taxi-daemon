package grpc

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/vulpemventures/liquid-taxi/internal/application"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
)

type adminServer struct {
	pb.UnimplementedAdminServer
	adminSvc application.AdminService
}

// NewAdminHandler returns a types implementing the pb.AdminServer grpc interface
func NewAdminHandler(adminSvc application.AdminService) pb.AdminServer {
	return &adminServer{
		adminSvc: adminSvc,
	}
}

func (a *adminServer) AddAsset(ctx context.Context, req *pb.AddAssetRequest) (*pb.AddAssetReply, error) {
	priceAmountAsDecimal := decimal.NewFromFloat(float64(req.GetPrice()))

	return &pb.AddAssetReply{}, a.adminSvc.AddAsset(
		ctx,
		req.GetAssetHash(),
		priceAmountAsDecimal,
	)
}

func (a *adminServer) ListTopups(ctx context.Context, req *pb.ListTopupsRequest) (*pb.ListTopupsReply, error) {
	topups, err := a.adminSvc.ListTopups(ctx, req.GetAssetHash())
	if err != nil {
		return nil, err
	}

	reply := make([]*pb.TopupInfo, 0)
	for _, t := range topups {
		reply = append(reply, t.ToProtobufTopupInfoMessage())
	}

	return &pb.ListTopupsReply{
		Topups: reply,
	}, nil
}

func (a *adminServer) ChangeSpread(ctx context.Context, req *pb.ChangeSpreadRequest) (*pb.ChangeSpreadReply, error) {
	// TODO implement change spread
	return nil, errors.New("Not implemented")
}
