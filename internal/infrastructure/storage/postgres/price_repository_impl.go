package postgresdb

import (
	"context"
	"database/sql"
	"github.com/shopspring/decimal"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/postgres/sqlc/queries"
)

type priceRepository struct {
	db      *sql.DB
	querier *queries.Queries
}

func NewPriceRepositoryImpl(db *sql.DB) domain.PriceRepository {
	return &priceRepository{
		db:      db,
		querier: queries.New(db),
	}
}

func (p *priceRepository) UpsertPrice(
	ctx context.Context,
	price *domain.Price,
) error {
	priceExist := true
	_, err := p.querier.GetPrice(ctx, price.AssetHash)
	if err != nil {
		if err == sql.ErrNoRows {
			priceExist = false
		} else {
			return err
		}
	}

	if priceExist {
		amount, _ := price.Amount.Float64()
		if err := p.querier.UpdatePrice(ctx, queries.UpdatePriceParams{
			Amount:     amount,
			BasisPoint: int32(price.BasisPoint),
			AssetHash:  price.AssetHash,
		}); err != nil {
			return err
		}
	} else {
		amount, _ := price.Amount.Float64()
		if _, err := p.querier.InsertPrice(ctx, queries.InsertPriceParams{
			Amount:     amount,
			BasisPoint: int32(price.BasisPoint),
			AssetHash:  price.AssetHash,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (p *priceRepository) GetPrice(
	ctx context.Context,
	assetHash string,
) (*domain.Price, error) {
	price, err := p.querier.GetPrice(ctx, assetHash)
	if err != nil {
		return nil, err
	}

	return &domain.Price{
		AssetHash:  price.AssetHash,
		Amount:     decimal.NewFromFloat(price.Amount),
		BasisPoint: uint32(price.BasisPoint),
	}, nil
}

func (p *priceRepository) GetAllPrices(
	ctx context.Context,
) ([]*domain.Price, error) {
	prices, err := p.querier.GetAllPrices(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	var pricesDomain []*domain.Price
	for _, price := range prices {
		pricesDomain = append(pricesDomain, &domain.Price{
			AssetHash:  price.AssetHash,
			Amount:     decimal.NewFromFloat(price.Amount),
			BasisPoint: uint32(price.BasisPoint),
		})
	}

	return pricesDomain, nil
}
