package domain

import "context"

type WhitelistRepository interface {
	AddScripts(ctx context.Context, scripts []string) (int, error)
	GetScripts(ctx context.Context) ([]string, error)
	RemoveScripts(ctx context.Context, scripts []string) (int, error)
}
