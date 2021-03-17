package grpc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	explorer "github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/admin"
	"github.com/vulpemventures/taxi-daemon/internal/application"
	"github.com/vulpemventures/taxi-daemon/internal/domain"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/inmemory"
)

const (
	explorerEndpoint = "http://localhost:3001"
	mnemonic         = "clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act soccer drive year mask"
)

var (
	words   = strings.Split(mnemonic, " ")
	regtest = network.Regtest
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

func newMockAdminHandler() (admin.AdminServer, error) {
	hdTreeRepo := inmemory.NewInmemoryHDTreeRepository(words, regtest)
	topupRepo := newMockedTopupRepository([]string{network.Liquid.AssetID})
	priceRepo := inmemory.NewInmemoryPriceRepository()
	lockRepo := inmemory.NewInmemoryLockRepository(20 * time.Second)
	explorerSvc, err := explorer.NewService(explorerEndpoint)
	if err != nil {
		return nil, err
	}

	cache := application.NewCache(application.NewCacheArgs{
		Explorer:         explorerSvc,
		HdTreeRepository: hdTreeRepo,
		LockRepository:   lockRepo,
		FragmentAsset:    regtest.AssetID,
		FragmentAmount:   20,
		Interval:         30 * time.Second,
	})

	args := application.NewAdminServiceArgs{
		HDTreeRepository: hdTreeRepo,
		TopupRepository:  topupRepo,
		PriceRepository:  priceRepo,
		LockRepository:   lockRepo,
		ExplorerService:  explorerSvc,
		Mnemonic:         words,
		Network:          regtest,
		Cache:            cache,
	}

	adminSvc := application.NewAdminService(args)

	return NewAdminHandler(adminSvc), nil
}

func TestListTopups(t *testing.T) {
	adminHandler, err := newMockAdminHandler()
	if err != nil {
		t.Fatal(err)
	}

	reply, err := adminHandler.ListTopups(context.Background(), &admin.ListTopupsRequest{
		AssetHash: "",
	})

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(reply.GetTopups()))
}
