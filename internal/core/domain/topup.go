package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TopupStatus is an enum type
type TopupStatus int32

const (
	// Requested is assigned to topup newly created
	Requested TopupStatus = iota
	// Completed is assigned to topup when the taxi PSET is created
	Completed
	// Settled is assigned to topup if the user broadcast the taxi PSET
	Settled
	// Expired is assigned to topup if it's not broadcasted within the expiration date
	Expired
)

var (
	// ErrTopupNotRequested is returned if attempting to complete a topup which
	// is not in Requested status.
	ErrTopupNotRequested = errors.New("topup must be requested to be completed")
	// ErrTopupNotCompleted is returned if attempting to settle a topup which
	// is not in Completed status.
	ErrTopupNotCompleted = errors.New("topup must be completed to be settled")
	// ErrTopupForbiddenExpiration is returned is attempting to set a topup as
	// expired before its expiration date.
	ErrTopupForbiddenExpiration = errors.New("topup cannot expire before expiration date")
)

// Topup represents the deal between taxi service and the user
// a topup lets a user to pay Liquid fees using a given assetHash (instead of LBTC)
type Topup struct {
	// Id is the unique Identifier
	ID string
	// Status is the current status of the Topup (Requested -> Completed -> Settled)
	Status TopupStatus
	// Partial is the base64 encoded taxi PSET (empty string until Completed status)
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

// NewTopupWithAsset is a factory function used to create an base topup
func NewTopupWithAsset(
	assetHash string,
	totalFee uint64,
	assetPrice decimal.Decimal,
	expireAt time.Time,
) *Topup {
	return &Topup{
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

func NewTopupWithPartial(partial string, expireAt time.Time) *Topup {
	return &Topup{
		ID:       uuid.New().String(),
		Status:   Requested,
		Partial:  partial,
		ExpireAt: expireAt,
	}
}

// Complete sets partial and assetAmount field + timestamp the complete time + update the status
func (t *Topup) Complete(partial string, assetAmount uint64, spreadAmount uint64) error {
	if t.Status != Requested {
		return ErrTopupNotRequested
	}

	t.CompletedAt = time.Now()
	t.Status = Completed
	t.Partial = partial
	t.AssetAmount = assetAmount
	t.SpreadAmount = spreadAmount
	return nil

}

// Settle timestamp the settle time and update the topup status
func (t *Topup) Settle() error {
	if t.Status != Completed {
		return ErrTopupNotCompleted
	}

	t.SettledAt = time.Now()
	t.Status = Settled
	return nil
}

func (t *Topup) Expire() error {
	if time.Now().Before(t.ExpireAt) {
		return ErrTopupForbiddenExpiration
	}

	t.Status = Expired
	return nil
}
