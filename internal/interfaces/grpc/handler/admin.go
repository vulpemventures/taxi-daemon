package handler

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/shopspring/decimal"
	pb "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/taxi-daemon/v1"
	"github.com/vulpemventures/taxi-daemon/internal/core/application"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type adminServer struct {
	pb.UnimplementedAdminServiceServer
	adminSvc application.AdminService
}

func NewAdminHandler(adminSvc application.AdminService) pb.AdminServiceServer {
	return &adminServer{adminSvc: adminSvc}
}

func (a *adminServer) AddAsset(
	ctx context.Context, req *pb.AddAssetRequest,
) (*pb.AddAssetResponse, error) {
	asset, err := parseAsset(req.GetAssetHash())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	price, err := parsePrice(req.GetPrice())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	bp := req.GetBasisPoints()

	if err := a.adminSvc.AddAsset(ctx, asset, price, bp); err != nil {
		return nil, err
	}
	return &pb.AddAssetResponse{}, nil
}

func (a *adminServer) AddWhitelistedAddresses(
	ctx context.Context, req *pb.AddWhitelistedAddressesRequest,
) (*pb.AddWhitelistedAddressesResponse, error) {
	addresses := req.GetAddresses()
	if err := a.adminSvc.AddWhitelistedAddresses(ctx, addresses); err != nil {
		return nil, err
	}
	return &pb.AddWhitelistedAddressesResponse{}, nil
}

func (a *adminServer) ListTopups(ctx context.Context, req *pb.ListTopupsRequest) (*pb.ListTopupsResponse, error) {
	assetFilter := req.GetAssetHash()
	topups, err := a.adminSvc.ListTopups(ctx, assetFilter)
	if err != nil {
		return nil, err
	}

	list := make([]*pb.TopupDetails, 0, len(topups))
	for _, t := range topups {
		list = append(list, toTopupDetailProto(t))
	}

	return &pb.ListTopupsResponse{Topups: list}, nil
}

func parseAsset(str string) (string, error) {
	if str == "" {
		return "", fmt.Errorf("missing asset hash")
	}
	buf, err := hex.DecodeString(str)
	if err != nil {
		return "", fmt.Errorf("invalid asset hash format, must be hex")
	}
	if len(buf) != 32 {
		return "", fmt.Errorf("invalid asset hash length, must be exactly 32 bytes")
	}
	return str, nil
}

func parsePrice(val float64) (dec decimal.Decimal, err error) {
	if val == 0 {
		err = fmt.Errorf("missing price")
		return
	}
	if val < 0 {
		err = fmt.Errorf("invalid price, must be a positive number")
		return
	}
	dec = decimal.NewFromFloat(val)
	return
}

func toTopupDetailProto(t *domain.Topup) *pb.TopupDetails {
	requestedAt := t.RequestedAt.Unix()
	if requestedAt < 0 {
		requestedAt = 0
	}
	completedAt := t.CompletedAt.Unix()
	if completedAt < 0 {
		completedAt = 0
	}
	settledAt := t.SettledAt.Unix()
	if settledAt < 0 {
		settledAt = 0
	}
	expiredAt := t.ExpireAt.Unix()
	if expiredAt < 0 {
		expiredAt = 0
	}
	times := &pb.Timestamps{
		RequestedAt: requestedAt,
		CompletedAt: completedAt,
		SettledAt:   settledAt,
		ExpiredAt:   expiredAt,
	}
	return &pb.TopupDetails{
		TopupId:     t.ID,
		Partial:     t.Partial,
		AssetHash:   t.AssetHash,
		AssetAmount: t.AssetAmount,
		AssetSpread: t.SpreadAmount,
		Status:      pb.TopupDetails_Status(t.Status),
		FeeAmount:   t.FeeAmount,
		Times:       times,
	}
}
