package domain

import "context"

type LockRepository interface {
	Add(ctx context.Context, keys []Lock) error
	Contains(ctx context.Context, key Lock) bool
	Remove(ctx context.Context, key Lock) error
}
