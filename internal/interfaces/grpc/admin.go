package grpc

import (
	"context"

	"github.com/vulpemventures/liquid-taxi/internal/application"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
)

type adminServer struct {
	pb.UnimplementedAdminServer
	adminSvc application.AdminService
}

var _ pb.AdminServer = (*adminServer)(nil)

func (a *adminServer) ListTopups(ctx context.Context, req *pb.ListTopupsRequest) (*pb.ListTopupsReply, error) {
	panic("not implemented")
}

func (a *adminServer) ChangeSpread(ctx context.Context, req *pb.ChangeSpreadRequest) (*pb.ChangeSpreadReply, error) {
	panic("not implemented")
}
