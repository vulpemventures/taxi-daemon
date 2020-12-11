package domain

import "time"

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
