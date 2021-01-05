package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
	"github.com/vulpemventures/liquid-taxi/internal/application"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
)

const (
	explorerEndpoint = "http://localhost:3001"
)

func newMockedTopupRepository(assets []string) domain.TopupRepository {
	fee := uint64(1000)
	price := decimal.NewFromInt(25000)
	expirationTime := time.Now().Add(10 * time.Minute)

	repo := inmemory.NewInmemoryTopupRepository()
	topups := make([]domain.Topup, len(assets), len(assets))
	for i, asset := range assets {
		topups[i] = domain.NewTopup(asset, fee, price, expirationTime)
	}

	ctx := context.Background()
	for _, t := range topups {
		repo.AddTopup(ctx, t)
	}

	return repo
}

func newMockAdminHandler() admin.AdminServer {
	hdTreeRepo := inmemory.NewInmemoryHDTreeRepository()
	topupRepo := newMockedTopupRepository([]string{network.Liquid.AssetID})
	priceRepo := inmemory.NewInmemoryPriceRepository()
	explorerSvc := explorer.NewService(explorerEndpoint)

	adminSvc := application.NewAdminService(hdTreeRepo, topupRepo, priceRepo, explorerSvc)

	return NewAdminHandler(adminSvc)
}

func TestListTopups(t *testing.T) {
	adminHandler := newMockAdminHandler()
	reply, err := adminHandler.ListTopups(context.Background(), &admin.ListTopupsRequest{
		AssetHash: "",
	})

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(reply.GetTopups()))
}
