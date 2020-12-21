package domain

import "context"

// LockRepository handles storage for Lock domain
type LockRepository interface {
	Lock(ctx context.Context, keys []Lock) error
	IsLocked(ctx context.Context, key Lock) bool
	Unlock(ctx context.Context, key Lock) error
}
