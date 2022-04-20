package domain

import "context"

type WhitelistRepository interface {
	AddAddresses(ctx context.Context, addresses []string) (int, error)
	GetAddresses(ctx context.Context) ([]string, error)
}
