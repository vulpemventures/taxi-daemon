package domain

import "context"

type PriceRepository interface {
	UpdateOrCreatePrice(ctx context.Context, price Price) error
	GetPrice(ctx context.Context, assetHash string) (*Price, bool, error)
	GetAllPrices(ctx context.Context) ([]Price, error)
}
