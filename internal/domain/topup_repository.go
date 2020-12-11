package domain

import "context"

type TopupRepository interface {
	AddTopup(ctx context.Context, topup Topup) error
	GetAllTopups(ctx context.Context) ([]Topup, error)
	SetRequested(ctx context.Context, topupId string) error
	SetCompleted(ctx context.Context, topupId string) error
	SetSettled(ctx context.Context, topupId string) error
}
