package domain

import "context"

// TopupRepository handles Topup storage
type TopupRepository interface {
	AddTopup(ctx context.Context, topup *Topup) error
	GetAllTopups(ctx context.Context) ([]*Topup, error)
	UpdateTopup(
		ctx context.Context, topupID string, updatefn func(*Topup) (*Topup, error),
	) error
}
