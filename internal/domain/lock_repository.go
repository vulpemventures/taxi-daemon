package domain

import (
	"context"
	"time"
)

// LockRepository handles storage for Lock domain
type LockRepository interface {
	// Lock an outpoint for a certain amount of time (i.e utxo TTL)
	Lock(ctx context.Context, lock []Lock) error
	IsLocked(ctx context.Context, lock Lock) bool
	Unlock(ctx context.Context, lock Lock) error
	GetLocks(ctx context.Context) ([]Lock, error)
	// GetConfigTTL used in Lock method
	GetConfigTTL(ctx context.Context) time.Duration
}
