package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/vulpemventures/liquid-taxi/pkg/utils"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
)

// TopupStatus is an enum type
type TopupStatus int32

const (
	// Requested is assigned to topup newly created
	Requested TopupStatus = 0
	// Completed is assigned to topup when the taxi PSET is created
	Completed TopupStatus = 1
	// Settled is assigned to topup if the user broadcast the taxi PSET
	Settled TopupStatus = 2
)

// Topup represents the deal between taxi service and the user
// a topup lets a user to pay Liquid fees using a given assetHash (instead of LBTC)
type Topup struct {
	// Id is the unique Identifier
	ID string
	// Status is the current status of the Topup (Requested -> Completed -> Settled)
	Status TopupStatus
	// Partial is the base64 encoded taxi PSET (empty string until Completed status)
	// A taxi PSET = 1 LBTC input from taxi account + 2 outputs (payout to taxi + fee)
	// the transaction MUST be signed
	Partial string
	// AssetHash is the asset used by the user to pay the fee
	AssetHash string
	// AssetAmount is the total amount of AssetHash to provide for the topup
	AssetAmount uint64
	// AssetPrice is the price used for the topup
	// price = number of satoshis of AssetHash to reach 1LBTC
	AssetPrice float64
	// FeeAmount is the total amount of bitcoin fee in the topup transaction
	FeeAmount uint64
	// RequestedAt is the moment when topup status was REQUESTED
	RequestedAt time.Time
	// CompletedAt is the moment when topup status was COMPLETED
	CompletedAt time.Time
	// SettledAt is the moment when topup status was SETTLED
	SettledAt time.Time
	// ExpireAt is the date where the utxo used in topup's transaction can be double-spent
	ExpireAt time.Time
}

// NewTopup is a factory function used to create an base topup
// The returned topup is not REQUESTED, the status is empty
func NewTopup(
	assetHash string,
	totalFee uint64,
	assetPrice float64,
	expireAt time.Time,
) Topup {
	return Topup{
		ID:          uuid.New().String(),
		Status:      Requested,
		Partial:     "",
		AssetHash:   assetHash,
		AssetPrice:  assetPrice,
		FeeAmount:   totalFee,
		ExpireAt:    expireAt,
		RequestedAt: time.Now(),
	}
}

// Complete sets partial and assetAmount field + timestamp the complete time + update the status
func (t *Topup) Complete(partial string, assetAmount uint64) {
	t.CompletedAt = time.Now()
	t.Status = Completed
	t.Partial = partial
	t.AssetAmount = assetAmount
}

// Settle timestamp the settle time and update the topup status
func (t *Topup) Settle() {
	t.SettledAt = time.Now()
	t.Status = Settled
}

// ToProtobufTopupMessage maps topup domain to pb.Topup message type
func (t *Topup) ToProtobufTopupMessage() *pb.Topup {
	return &pb.Topup{
		TopupId:     t.ID,
		Partial:     t.Partial,
		AssetHash:   t.AssetHash,
		AssetAmount: t.AssetAmount,
		AssetPrice:  float32(t.AssetPrice),
	}
}

// ToProtobufTimestampsMessage maps topup domain to pb.Timestamps message type
func (t *Topup) ToProtobufTimestampsMessage() *pb.Timestamps {
	return &pb.Timestamps{
		RequestedAt: utils.MakeMillisecondTimestamp(t.RequestedAt),
		CompletedAt: utils.MakeMillisecondTimestamp(t.CompletedAt),
		SettledAt:   utils.MakeMillisecondTimestamp(t.SettledAt),
	}
}

// ToProtobufTopupInfoMessage maps topup domain to pb.TopupInfo message type
func (t *Topup) ToProtobufTopupInfoMessage() *pb.TopupInfo {
	return &pb.TopupInfo{
		Topup:           t.ToProtobufTopupMessage(),
		Status:          pb.TopupInfo_Status(t.Status),
		FeeAmount:       t.FeeAmount,
		MillisatPerByte: 0,
		Times:           t.ToProtobufTimestampsMessage(),
	}
}
