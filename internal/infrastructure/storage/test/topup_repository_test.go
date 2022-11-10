package db_test

import (
	"encoding/base64"
	"fmt"
	postgresdb "github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/postgres"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	dbbadger "github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/badger"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/inmemory"
)

var (
	topup *domain.Topup
)

func TestTopupRepository(t *testing.T) {
	repositories, err := newTopupRepositories()
	require.NoError(t, err)

	for name, repo := range repositories {
		t.Run(name, func(t *testing.T) {
			testTopupRepository(t, repo)
		})
	}
}

func testTopupRepository(t *testing.T, repo domain.TopupRepository) {
	testAddTopup(t, repo)
	testUpdateTopup(t, repo)
}

func testAddTopup(t *testing.T, repo domain.TopupRepository) {
	topups, err := repo.GetAllTopups(ctx)
	require.NoError(t, err)
	require.Empty(t, topups)

	topup = randomTopup()
	err = repo.AddTopup(ctx, topup)
	require.NoError(t, err)

	topups, err = repo.GetAllTopups(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, topups)
	require.Len(t, topups, 1)
	require.Exactly(t, *topup, *topups[0])

	err = repo.AddTopup(ctx, topup)
	require.Error(t, err)

	anotherTopup := randomTopup()
	err = repo.AddTopup(ctx, anotherTopup)
	require.NoError(t, err)

	topups, err = repo.GetAllTopups(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, topups)
	require.Len(t, topups, 2)
}

func testUpdateTopup(t *testing.T, repo domain.TopupRepository) {
	topupId := uuid.New().String()
	err := repo.UpdateTopup(
		ctx, topupId, func(t *domain.Topup) (*domain.Topup, error) {
			return t, nil
		},
	)
	require.Error(t, err)

	expectedErr := fmt.Errorf("something went wrong")
	err = repo.UpdateTopup(
		ctx, topup.ID, func(t *domain.Topup) (*domain.Topup, error) {
			return nil, expectedErr
		},
	)
	require.EqualError(t, err, expectedErr.Error())

	err = repo.UpdateTopup(
		ctx, topup.ID, func(t *domain.Topup) (*domain.Topup, error) {
			t.RequestedAt = time.Now()
			t.ExpireAt = time.Now().Add(5 * time.Minute)
			return t, nil
		},
	)
	require.NoError(t, err)

	topups, err := repo.GetAllTopups(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, topups)
	found := false
	for _, tt := range topups {
		if tt.ID == topup.ID {
			require.False(t, tt.RequestedAt.IsZero())
			found = true
		} else {
			require.True(t, tt.RequestedAt.IsZero())
		}
	}
	require.True(t, found)
}

func newTopupRepositories() (map[string]domain.TopupRepository, error) {
	inmemoryRepoManager := inmemory.NewRepoManager()
	badgerRepoManager, err := dbbadger.NewRepoManager("", nil)
	if err != nil {
		return nil, err
	}

	pgRepoManager, err := postgresdb.NewRepoManager(postgresdb.DbConfig{
		DbUser:             "root",
		DbPassword:         "secret",
		DbHost:             "127.0.0.1",
		DbPort:             5432,
		DbName:             "taxid-db-test",
		MigrationSourceURL: "file://../postgres/migration",
	})
	if err != nil {
		return nil, err
	}

	return map[string]domain.TopupRepository{
		"inmemory": inmemoryRepoManager.TopupRepository(),
		"badger":   badgerRepoManager.TopupRepository(),
		"postgres": pgRepoManager.TopupRepository(),
	}, nil
}

func randomTopup() *domain.Topup {
	return &domain.Topup{
		ID:           uuid.New().String(),
		Status:       randomStatus(),
		Partial:      randomPartial(),
		AssetHash:    randomAsset(),
		AssetAmount:  uint64(randomIntInRange(10000, 100000)),
		AssetPrice:   randomAmount(),
		FeeAmount:    uint64(randomIntInRange(300, 1000)),
		SpreadAmount: uint64(randomIntInRange(1000, 10000)),
	}
}

func randomStatus() domain.TopupStatus {
	return domain.TopupStatus(randomIntInRange(0, 2))
}

func randomPartial() string {
	return base64.StdEncoding.EncodeToString(randomBytes(100))
}
