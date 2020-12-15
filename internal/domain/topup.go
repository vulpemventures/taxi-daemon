package domain

import (
	"time"

	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
)

type TopupStatus int

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
