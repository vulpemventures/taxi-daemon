package grpc

import (
	"context"
	"errors"

	"github.com/vulpemventures/liquid-taxi/internal/application"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
)

type adminServer struct {
	pb.UnimplementedAdminServer
	adminSvc application.AdminService
}

func NewAdminHandler(adminSvc application.AdminService) pb.AdminServer {
	return &adminServer{
		adminSvc: adminSvc,
	}
}

func (a *adminServer) ListTopups(ctx context.Context, req *pb.ListTopupsRequest) (*pb.ListTopupsReply, error) {
	_, err := a.adminSvc.ListTopups(req.GetAssetHash())
	if err != nil {
		return nil, err
	}

	// TODO return listTopupsReply
	return nil, errors.New("Not implemented")
}

func (a *adminServer) ChangeSpread(ctx context.Context, req *pb.ChangeSpreadRequest) (*pb.ChangeSpreadReply, error) {
	// TODO implement change spread
	return nil, errors.New("Not implemented")
}
