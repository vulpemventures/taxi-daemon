package domain

import "errors"

var (
	// ErrTopupNeedToBeRequested is returned by Topup.Complete if the topup's status is not REQUESTED
	ErrTopupNeedToBeRequested = errors.New("topup needs to be requested")
	// ErrTopupNeedToBeCompleted is returned by Topup.Settle if the topup's status is not COMPLETED
	ErrTopupNeedToBeCompleted = errors.New("topup needs to be completed")
)
