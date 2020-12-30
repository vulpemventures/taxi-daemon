package domain

import "context"

type HDTreeRepository interface {
	GetOrCreateHDTree(ctx context.Context) (*HDTree, error)
	GetNewFundingAddresses(ctx context.Context, numOfAddresses uint) ([]string, error)
	GetNewPayoutAddresses(ctx context.Context, numOfAddresses uint) ([]string, error)
}
