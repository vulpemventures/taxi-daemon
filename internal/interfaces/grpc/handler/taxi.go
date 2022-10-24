package handler

import (
	"context"
	"fmt"

	"github.com/vulpemventures/go-elements/psetv2"
	pb "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/taxi/v1"
	"github.com/vulpemventures/taxi-daemon/internal/core/application"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type taxiHandler struct {
	svc application.TaxiService
}

func NewTaxiHandler(taxiSvc application.TaxiService) pb.TaxiServiceServer {
	return &taxiHandler{taxiSvc}
}

func (h *taxiHandler) ListAssets(
	ctx context.Context, _ *pb.ListAssetsRequest,
) (*pb.ListAssetsResponse, error) {
	assets, err := h.svc.ListAssets(ctx)
	if err != nil {
		return nil, err
	}

	list := make([]*pb.AssetDetails, 0, len(assets))
	for _, a := range assets {
		list = append(list, toAssetDetailProto(a))
	}

	return &pb.ListAssetsResponse{Assets: list}, nil
}

func (h *taxiHandler) TopupWithAsset(
	ctx context.Context, req *pb.TopupWithAssetRequest,
) (*pb.TopupWithAssetResponse, error) {
	asset, err := parseAsset(req.GetAssetHash())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	estimatedSize, err := parseEstimatedSize(req.GetEstimatedTxSize())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	millisatPerByte := req.GetMillisatPerByte()

	topup, selectedUtxos, err := h.svc.TopupWithAsset(
		ctx, asset, estimatedSize, millisatPerByte,
	)
	if err != nil {
		return nil, err
	}

	blindingData := make([]*pb.BlindingData, 0, len(selectedUtxos))
	for _, u := range selectedUtxos {
		blindingData = append(blindingData, &pb.BlindingData{
			Asset:        u.Asset(),
			Value:        u.Value(),
			AssetBlinder: u.AssetBlinder(),
			ValueBlinder: u.ValueBlinder(),
		})
	}

	return &pb.TopupWithAssetResponse{
		Topup:          toTopupProto(topup),
		Expiry:         topup.ExpireAt.Unix(),
		AssetHash:      topup.AssetHash,
		AssetAmount:    topup.AssetAmount,
		AssetSpread:    topup.SpreadAmount,
		InBlindingData: blindingData,
	}, nil
}

func (h *taxiHandler) TopupWithKnownPayout(
	ctx context.Context, req *pb.TopupWithKnownPayoutRequest,
) (*pb.TopupWithKnownPayoutResponse, error) {
	psetBase64, err := parsePartial(req.GetPartial())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	millisatPerByte := req.GetMillisatPerByte()

	topup, err := h.svc.TopupWithKnownPayout(
		ctx,
		psetBase64,
		millisatPerByte,
		application.UnblindedInput{
			Index:         int(req.GetUnblindedInput().GetIndex()),
			Asset:         req.GetUnblindedInput().GetAsset(),
			Amount:        int(req.GetUnblindedInput().GetAmount()),
			AssetBlinder:  req.GetUnblindedInput().GetAssetBlinder(),
			AmountBlinder: req.GetUnblindedInput().GetAmountBlinder(),
		})
	if err != nil {
		return nil, err
	}

	return &pb.TopupWithKnownPayoutResponse{
		Topup:  toTopupProto(topup),
		Expiry: topup.ExpireAt.Unix(),
	}, nil
}

func parseEstimatedSize(size uint64) (uint64, error) {
	if size == 0 {
		return 0, fmt.Errorf("missing estimated tx size")
	}
	if int(size) < 0 {
		return 0, fmt.Errorf("invalid size, must be a positive value")
	}
	return size, nil
}

func toAssetDetailProto(p *domain.Price) *pb.AssetDetails {
	amount, _ := p.Amount.Float64()
	return &pb.AssetDetails{
		AssetHash:  p.AssetHash,
		AssetPrice: amount,
		BasisPoint: p.BasisPoint,
	}
}

func toTopupProto(t *domain.Topup) *pb.Topup {
	return &pb.Topup{
		TopupId: t.ID,
		Partial: t.Partial,
	}
}

func parsePartial(str string) (string, error) {
	if str == "" {
		return "", fmt.Errorf("missing partial transaction")
	}
	if _, err := psetv2.NewPsetFromBase64(str); err != nil {
		return "", fmt.Errorf("invalid partial transaction: %s", err.Error())
	}
	return str, nil
}
