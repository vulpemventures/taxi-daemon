package domain

import "context"

type UtxoKeyRepository interface {
	AddKeys(ctx context.Context, keys []UtxoKey) error
	Contains(ctx context.Context, key UtxoKey) bool
	Remove(ctx context.Context, key UtxoKey) error
}
