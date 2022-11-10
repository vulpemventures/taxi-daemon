package postgresdb

import (
	"context"
	"database/sql"
	"github.com/lib/pq"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/postgres/sqlc/queries"
)

const pgUniqueKeyErrorCode = "23505"

type whitelistRepository struct {
	db      *sql.DB
	querier *queries.Queries
}

func NewWhitelistRepositoryImpl(db *sql.DB) domain.WhitelistRepository {
	return &whitelistRepository{
		db:      db,
		querier: queries.New(db),
	}
}

func (w *whitelistRepository) AddScripts(
	ctx context.Context,
	scripts []string,
) (int, error) {
	count := 0
	for _, script := range scripts {
		if _, err := w.querier.InsertWhitelist(ctx, script); err != nil {
			if v, ok := err.(*pq.Error); ok {
				if v.Code == pgUniqueKeyErrorCode {
					continue
				}
			}
			return 0, err
		}

		count++
	}

	return count, nil
}

func (w *whitelistRepository) GetScripts(
	ctx context.Context,
) ([]string, error) {
	return w.querier.GetWhitelists(ctx)
}

func (w *whitelistRepository) RemoveScripts(
	ctx context.Context,
	scripts []string,
) (int, error) {
	for _, script := range scripts {
		if err := w.querier.DeleteWhitelist(ctx, script); err != nil {
			return 0, err
		}
	}

	return len(scripts), nil
}
