package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/vulpemventures/liquid-taxi/pkg/utils"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
)

type TopupStatus int32

const (
	Requested TopupStatus = 0
	Completed TopupStatus = 1
	Settled   TopupStatus = 2
)

type Topup struct {
	Id          string
	Status      TopupStatus
	Partial     string
	AssetHash   string
	AssetAmount uint64
	AssetPrice  float64
	FeeAmount   uint64
	BasisPoint  int32
	RequestedAt time.Time
	CompletedAt time.Time
	SettledAt   time.Time
	ExpireAt    time.Time
}

func NewTopup(
	assetHash string,
	totalFee uint64,
	assetPrice float64,
	expireAt time.Time,
) Topup {
	return Topup{
		Id:         uuid.New().String(),
		Partial:    "",
		AssetHash:  assetHash,
		AssetPrice: assetPrice,
		FeeAmount:  totalFee,
		ExpireAt:   expireAt,
	}
}

func (t *Topup) ToProtobufTopupMessage() *pb.Topup {
	return &pb.Topup{
		TopupId:     t.Id,
		Partial:     t.Partial,
		AssetHash:   t.AssetHash,
		AssetAmount: t.AssetAmount,
		AssetPrice:  float32(t.AssetPrice),
		BasisPoint:  t.BasisPoint,
	}
}

func (t *Topup) ToProtobufTimestampsMessage() *pb.Timestamps {
	return &pb.Timestamps{
		RequestedAt: utils.MakeMillisecondTimestamp(t.RequestedAt),
		CompletedAt: utils.MakeMillisecondTimestamp(t.CompletedAt),
		SettledAt:   utils.MakeMillisecondTimestamp(t.SettledAt),
	}
}

func (t *Topup) ToProtobufTopupInfoMessage() *pb.TopupInfo {
	return &pb.TopupInfo{
		Topup:           t.ToProtobufTopupMessage(),
		Status:          pb.TopupInfo_Status(t.Status),
		FeeAmount:       t.FeeAmount,
		MillisatPerByte: 0,
		Times:           t.ToProtobufTimestampsMessage(),
	}
}
