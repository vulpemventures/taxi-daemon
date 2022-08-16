package domain

import "context"

// PriceRepository handles Price storage
type PriceRepository interface {
	UpsertPrice(ctx context.Context, price *Price) error
	GetPrice(ctx context.Context, assetHash string) (*Price, error)
	GetAllPrices(ctx context.Context) ([]*Price, error)
}
