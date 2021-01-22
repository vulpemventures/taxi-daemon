package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	admin "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/admin"
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
	// AssetPrice is the price AssetHash/LBTC used for the topup
	AssetPrice decimal.Decimal
	// FeeAmount is the total amount of bitcoin fee in the topup transaction
	FeeAmount uint64
	// RequestedAt is the moment when topup status was REQUESTED
	RequestedAt time.Time
	// CompletedAt is the moment when topup status was COMPLETED
	CompletedAt time.Time
	// SettledAt is the moment when topup status was SETTLED
	SettledAt time.Time
	// ExpireAt is the date where the utxo used in topup's transaction can be used for other topups
	ExpireAt time.Time
	// SpreadAmount is the amount of asset paid as taxi service fee
	SpreadAmount uint64
}

// NewTopup is a factory function used to create an base topup
func NewTopup(
	assetHash string,
	totalFee uint64,
	assetPrice decimal.Decimal,
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
func (t *Topup) Complete(partial string, assetAmount uint64, spreadAmount uint64) error {
	if t.Status == Requested {
		t.CompletedAt = time.Now()
		t.Status = Completed
		t.Partial = partial
		t.AssetAmount = assetAmount
		t.SpreadAmount = spreadAmount
		return nil
	}

	return ErrTopupNeedToBeRequested
}

// Settle timestamp the settle time and update the topup status
func (t *Topup) Settle() error {
	if t.Status == Completed {
		t.SettledAt = time.Now()
		t.Status = Settled
		return nil
	}

	return ErrTopupNeedToBeCompleted
}

// ToTopupMessage maps topup domain to taxi.Topup message type
func (t *Topup) ToTopupMessage() *pb.Topup {
	return &pb.Topup{
		TopupId:     t.ID,
		Partial:     t.Partial,
		AssetHash:   t.AssetHash,
		AssetAmount: t.AssetAmount,
		AssetSpread: t.SpreadAmount,
	}
}

// ToTopupDetailsMessage maps topup domain to admin.TopupDetails message type
func (t *Topup) ToTopupDetailsMessage() *admin.TopupDetails {
	return &admin.TopupDetails{
		TopupId:     t.ID,
		Partial:     t.Partial,
		AssetHash:   t.AssetHash,
		AssetAmount: t.AssetAmount,
		AssetSpread: t.SpreadAmount,
		Status:      admin.TopupDetails_Status(t.Status),
		FeeAmount:   t.FeeAmount,
		Times:       t.toProtobufTimestampsMessage(),
		Txid:        "",
	}
}

func (t *Topup) toProtobufTimestampsMessage() *admin.Timestamps {
	return &admin.Timestamps{
		RequestedAt: uint64(t.RequestedAt.Unix()),
		CompletedAt: uint64(t.CompletedAt.Unix()),
		SettledAt:   uint64(t.SettledAt.Unix()),
		ExpiredAt:   uint64(t.ExpireAt.Unix()),
	}
}
