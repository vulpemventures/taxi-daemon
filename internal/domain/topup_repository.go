package domain

import "context"

// TopupRepository handles Topup storage
type TopupRepository interface {
	AddTopup(ctx context.Context, topup Topup) error
	GetAllTopups(ctx context.Context) ([]Topup, error)
	MarkAsCompleted(ctx context.Context, topupID string, partial string, assetAmount uint64) (*Topup, error)
	MarkAsSettled(ctx context.Context, topupID string, txID string) (*Topup, error)
}
