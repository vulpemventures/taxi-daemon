package postgresdb

import (
	"context"
	"database/sql"
	"github.com/shopspring/decimal"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/postgres/sqlc/queries"
	"time"
)

type topupRepository struct {
	db      *sql.DB
	querier *queries.Queries
}

func NewTopupRepositoryImpl(db *sql.DB) domain.TopupRepository {
	return &topupRepository{
		db:      db,
		querier: queries.New(db),
	}
}

func (t *topupRepository) AddTopup(
	ctx context.Context,
	topup *domain.Topup,
) error {
	partial := sql.NullString{
		String: "",
		Valid:  false,
	}
	if topup.Partial != "" {
		partial = sql.NullString{
			String: topup.Partial,
			Valid:  true,
		}
	}
	assetPrice, _ := topup.AssetPrice.Float64()
	requestedAt := sql.NullTime{
		Time:  time.Time{},
		Valid: false,
	}
	if !topup.RequestedAt.IsZero() {
		requestedAt = sql.NullTime{
			Time:  topup.RequestedAt,
			Valid: true,
		}
	}
	completedAt := sql.NullTime{
		Time:  time.Time{},
		Valid: false,
	}
	if !topup.CompletedAt.IsZero() {
		completedAt = sql.NullTime{
			Time:  topup.CompletedAt,
			Valid: true,
		}
	}
	settledAt := sql.NullTime{
		Time: time.Time{},
	}
	if !topup.SettledAt.IsZero() {
		settledAt = sql.NullTime{
			Time:  topup.SettledAt,
			Valid: true,
		}
	}
	expireAt := sql.NullTime{
		Time: time.Time{},
	}
	if !topup.ExpireAt.IsZero() {
		expireAt = sql.NullTime{
			Time:  topup.ExpireAt,
			Valid: true,
		}
	}
	if _, err := t.querier.InsertTopup(ctx, queries.InsertTopupParams{
		ID:           topup.ID,
		Status:       int32(topup.Status),
		PartialTx:    partial,
		AssetHash:    topup.AssetHash,
		AssetAmount:  int32(topup.AssetAmount),
		AssetPrice:   assetPrice,
		FeeAmount:    int32(topup.FeeAmount),
		RequestedAt:  requestedAt,
		CompletedAt:  completedAt,
		SettledAt:    settledAt,
		ExpireAt:     expireAt,
		SpreadAmount: int32(topup.SpreadAmount),
	}); err != nil {
		return err
	}

	return nil
}

func (t *topupRepository) GetAllTopups(
	ctx context.Context,
) ([]*domain.Topup, error) {
	topups, err := t.querier.GetAllTopups(ctx)
	if err != nil {
		return nil, err
	}

	var result []*domain.Topup
	for _, topup := range topups {
		requestedAt := time.Time{}
		if topup.RequestedAt.Valid {
			requestedAt = topup.RequestedAt.Time
		}
		completedAt := time.Time{}
		if topup.CompletedAt.Valid {
			completedAt = topup.CompletedAt.Time
		}
		settledAt := time.Time{}
		if topup.SettledAt.Valid {
			settledAt = topup.SettledAt.Time
		}
		expireAt := time.Time{}
		if topup.ExpireAt.Valid {
			expireAt = topup.ExpireAt.Time
		}
		result = append(result, &domain.Topup{
			ID:           topup.ID,
			Status:       domain.TopupStatus(topup.Status),
			Partial:      topup.PartialTx.String,
			AssetHash:    topup.AssetHash,
			AssetAmount:  uint64(topup.AssetAmount),
			AssetPrice:   decimal.NewFromFloat(topup.AssetPrice),
			FeeAmount:    uint64(topup.FeeAmount),
			RequestedAt:  requestedAt,
			CompletedAt:  completedAt,
			SettledAt:    settledAt,
			ExpireAt:     expireAt,
			SpreadAmount: uint64(topup.SpreadAmount),
		})
	}

	return result, nil
}

func (t *topupRepository) UpdateTopup(
	ctx context.Context,
	topupID string,
	updateFn func(*domain.Topup) (*domain.Topup, error),
) error {
	topup, err := t.querier.GetTopup(ctx, topupID)
	if err != nil {
		return err
	}

	partialTx := ""
	if topup.PartialTx.Valid {
		partialTx = topup.PartialTx.String
	}
	requested := time.Time{}
	if topup.CompletedAt.Valid {
		requested = topup.RequestedAt.Time
	}
	completed := time.Time{}
	if topup.CompletedAt.Valid {
		completed = topup.CompletedAt.Time
	}
	settled := time.Time{}
	if topup.SettledAt.Valid {
		settled = topup.SettledAt.Time
	}
	expire := time.Time{}
	if topup.ExpireAt.Valid {
		expire = topup.ExpireAt.Time
	}
	updatedTopup, err := updateFn(
		&domain.Topup{
			ID:           topup.ID,
			Status:       domain.TopupStatus(topup.Status),
			Partial:      partialTx,
			AssetHash:    topup.AssetHash,
			AssetAmount:  uint64(topup.AssetAmount),
			AssetPrice:   decimal.NewFromFloat(topup.AssetPrice),
			FeeAmount:    uint64(topup.FeeAmount),
			RequestedAt:  requested,
			CompletedAt:  completed,
			SettledAt:    settled,
			ExpireAt:     expire,
			SpreadAmount: uint64(topup.SpreadAmount),
		},
	)
	if err != nil {
		return err
	}

	partial := sql.NullString{
		String: "",
		Valid:  false,
	}
	if updatedTopup.Partial != "" {
		partial = sql.NullString{
			String: updatedTopup.Partial,
			Valid:  true,
		}
	}
	assetPrice, _ := updatedTopup.AssetPrice.Float64()
	requestedAt := sql.NullTime{
		Time:  time.Time{},
		Valid: false,
	}
	if !updatedTopup.RequestedAt.IsZero() {
		requestedAt = sql.NullTime{
			Time:  updatedTopup.RequestedAt,
			Valid: true,
		}
	}
	completedAt := sql.NullTime{
		Time:  time.Time{},
		Valid: false,
	}
	if !updatedTopup.CompletedAt.IsZero() {
		completedAt = sql.NullTime{
			Time:  updatedTopup.CompletedAt,
			Valid: true,
		}
	}
	settledAt := sql.NullTime{
		Time: time.Time{},
	}
	if !updatedTopup.SettledAt.IsZero() {
		settledAt = sql.NullTime{
			Time:  updatedTopup.SettledAt,
			Valid: true,
		}
	}
	expireAt := sql.NullTime{
		Time: time.Time{},
	}
	if !updatedTopup.ExpireAt.IsZero() {
		expireAt = sql.NullTime{
			Time:  updatedTopup.ExpireAt,
			Valid: true,
		}
	}
	return t.querier.UpdateTopup(ctx, queries.UpdateTopupParams{
		ID:           updatedTopup.ID,
		Status:       int32(updatedTopup.Status),
		PartialTx:    partial,
		AssetHash:    updatedTopup.AssetHash,
		AssetAmount:  int32(updatedTopup.AssetAmount),
		AssetPrice:   assetPrice,
		FeeAmount:    int32(updatedTopup.FeeAmount),
		RequestedAt:  requestedAt,
		CompletedAt:  completedAt,
		SettledAt:    settledAt,
		ExpireAt:     expireAt,
		SpreadAmount: int32(updatedTopup.SpreadAmount),
	})
}
