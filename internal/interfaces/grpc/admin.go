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

func (a *adminServer) ListFundingAddresses(ctx context.Context, req *admin.ListFundingAddressesRequest) (*admin.ListFundingAddressesReply, error) {
	list, err := a.adminSvc.ListFundingAddresses(ctx, uint(req.GetStart()), uint(req.GetNumber()))
	if err != nil {
		return nil, err
	}

	addrInfoList := make([]*admin.AddressInfo, len(list), len(list))
	for i, info := range list {
		addrInfoList[i] = &admin.AddressInfo{
			Address:        info.Address,
			DerivationPath: info.DerivationPath,
			Index:          int64(info.Index),
			PublicKey:      info.PublicKey,
		}
	}
	return &admin.ListFundingAddressesReply{
		Addresses: addrInfoList,
	}, nil
}

func (a *adminServer) LatestFundingAddressIndex(ctx context.Context, _ *admin.LatestFundingAddressIndexRequest) (*admin.LatestFundingAddressIndexReply, error) {
	latestIndex, err := a.adminSvc.LatestFundingAddressIndex(ctx)
	if err != nil {
		return nil, err
	}
	return &admin.LatestFundingAddressIndexReply{
		Index: int64(latestIndex),
	}, nil
}

func (a *adminServer) Balance(ctx context.Context, _ *admin.BalanceRequest) (*admin.BalanceReply, error) {
	balanceInfosMap, err := a.adminSvc.Balance(ctx)
	if err != nil {
		return nil, err
	}

	protobufMap := make(map[string]*admin.BalanceInfo, len(balanceInfosMap))

	for asset, balanceInfo := range balanceInfosMap {
		protobufMap[asset] = &admin.BalanceInfo{
			TotalBalance:       balanceInfo.TotalBalance,
			ConfirmedBalance:   balanceInfo.ConfirmedBalance,
			UnconfirmedBalance: balanceInfo.UnconfirmedBalance,
		}
	}

	return &admin.BalanceReply{
		Balance: protobufMap,
	}, nil
}

func (a *adminServer) SendToMany(context.Context, *admin.SendToManyRequest) (*admin.SendToManyReply, error) {
	panic("not implemented")
}
