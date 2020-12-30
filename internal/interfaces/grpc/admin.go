package grpc

import (
	"context"

	"github.com/shopspring/decimal"
	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
	"github.com/vulpemventures/liquid-taxi/internal/application"
)

type adminServer struct {
	admin.UnimplementedAdminServer
	adminSvc application.AdminService
}

// NewAdminHandler returns a types implementing the admin.AdminServer grpc interface
func NewAdminHandler(adminSvc application.AdminService) admin.AdminServer {
	return &adminServer{
		adminSvc: adminSvc,
	}
}

func (a *adminServer) AddAsset(ctx context.Context, req *admin.AddAssetRequest) (*admin.AddAssetReply, error) {
	priceAmountAsDecimal := decimal.NewFromFloat(float64(req.GetPrice()))

	return &admin.AddAssetReply{}, a.adminSvc.AddAsset(
		ctx,
		req.GetAssetHash(),
		priceAmountAsDecimal,
	)
}

func (a *adminServer) FundingAddress(ctx context.Context, req *admin.FundingAddressRequest) (*admin.FundingAddressReply, error) {
	addresses, err := a.adminSvc.GetFundingAddresses(ctx, uint(req.GetNumOfAddresses()))
	if err != nil {
		return nil, err
	}

	return &admin.FundingAddressReply{
		Addresses: addresses,
	}, nil
}

func (a *adminServer) ListTopups(ctx context.Context, req *admin.ListTopupsRequest) (*admin.ListTopupsReply, error) {
	topups, err := a.adminSvc.ListTopups(ctx, req.GetAssetHash())
	if err != nil {
		return nil, err
	}

	reply := make([]*admin.TopupDetails, 0)
	for _, t := range topups {
		reply = append(reply, t.ToTopupDetailsMessage())
	}

	return &admin.ListTopupsReply{
		Topups: reply,
	}, nil
}

func (a *adminServer) ListAddresses(context.Context, *admin.ListAddressesRequest) (*admin.ListAddressesReply, error) {
	panic("not implemented")
}

func (a *adminServer) Balance(context.Context, *admin.BalanceRequest) (*admin.BalanceReply, error) {
	panic("not implemented")
}

func (a *adminServer) SendToMany(context.Context, *admin.SendToManyRequest) (*admin.SendToManyReply, error) {
	panic("not implemented")
}
