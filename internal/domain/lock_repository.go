package domain

import "context"

// LockRepository handles storage for Lock domain
type LockRepository interface {
	Add(ctx context.Context, keys []Lock) error
	Contains(ctx context.Context, key Lock) bool
	Remove(ctx context.Context, key Lock) error
}
