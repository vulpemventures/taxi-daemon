package domain

import "context"

// HDTreeRepository is the repository interface for HDTree domain
type HDTreeRepository interface {
	GetOrCreateHDTree(ctx context.Context) (*HDTree, error)
	GetNewFundingAddresses(ctx context.Context, numOfAddresses uint) ([]string, error)
	GetNewPayoutAddresses(ctx context.Context, numOfAddresses uint) ([]string, error)
	GetBlindingPrivateKey(ctx context.Context, witnessScript string) (string, error)
}
