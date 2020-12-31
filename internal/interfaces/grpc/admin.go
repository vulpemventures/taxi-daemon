package grpc

import (
	"context"
	"encoding/hex"

	"github.com/shopspring/decimal"
	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
	"github.com/vulpemventures/liquid-taxi/internal/application"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
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

func (a *adminServer) Balance(context.Context, *admin.BalanceRequest) (*admin.BalanceReply, error) {
	panic("not implemented")
}

func (a *adminServer) SendToMany(ctx context.Context, req *admin.SendToManyRequest) (*admin.SendToManyReply, error) {
	outputs := make([]wallet.Receiver, len(req.Outputs), len(req.Outputs))
	for i, out := range req.Outputs {
		outputs[i] = wallet.Receiver{
			Address: out.Address,
			Asset:   out.Asset,
			Amount:  uint64(out.Value),
		}
	}

	txHex, err := a.adminSvc.SendToMany(ctx, outputs, req.MillisatPerByte, req.Push)
	if err != nil {
		return nil, err
	}

	bytes, err := hex.DecodeString(txHex)

	return &admin.SendToManyReply{
		RawTx: bytes,
	}, err
}
