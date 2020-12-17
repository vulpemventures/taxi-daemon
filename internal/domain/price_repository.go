package domain

import "context"

// PriceRepository handles Price storage
type PriceRepository interface {
	UpdateOrCreatePrice(ctx context.Context, price Price) error
	GetPrice(ctx context.Context, assetHash string) (*Price, bool, error)
	GetAllPrices(ctx context.Context) ([]Price, error)
}
