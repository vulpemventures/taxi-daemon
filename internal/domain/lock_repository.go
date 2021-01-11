package domain

import "context"

// LockRepository handles storage for Lock domain
type LockRepository interface {
	Lock(ctx context.Context, locks []Lock) error
	IsLocked(ctx context.Context, lock Lock) bool
	Unlock(ctx context.Context, lock Lock) error
	GetLocks(ctx context.Context) ([]Lock, error)
}
