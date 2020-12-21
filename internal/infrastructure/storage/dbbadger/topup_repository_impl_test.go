package dbbadger

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

const (
	testDataDir = "./test-datadir"
)

func TestAddTopup(t *testing.T) {
	dirManagementForTest(t)

	repo, err := NewBadgerTopupRepository(testDataDir, nil)
	if err != nil {
		t.Error(err)
	}

	topups, err := repo.GetAllTopups(context.Background())
	if err != nil {
		t.Error(err)
	}

	err = repo.AddTopup(context.Background(), domain.NewTopup("aa", 0, 1.54, time.Now()))

	assert.Nil(t, err)

	topupsBis, err := repo.GetAllTopups(context.Background())
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 1, len(topupsBis)-len(topups))
}

func TestGetAllTopups(t *testing.T) {
	dirManagementForTest(t)

	repo, err := NewBadgerTopupRepository(testDataDir, nil)
	if err != nil {
		t.Error(err)
	}

	err = repo.AddTopup(context.Background(), domain.NewTopup("aa", 0, 1.54, time.Now()))
	if err != nil {
		t.Error(err)
	}
	err = repo.AddTopup(context.Background(), domain.NewTopup("aa", 0, 1.54, time.Now()))
	if err != nil {
		t.Error(err)
	}
	err = repo.AddTopup(context.Background(), domain.NewTopup("aa", 0, 1.54, time.Now()))
	if err != nil {
		t.Error(err)
	}

	topups, err := repo.GetAllTopups(context.Background())
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, 3, len(topups))
}

func TestSetCompleted(t *testing.T) {
	dirManagementForTest(t)

	repo, err := NewBadgerTopupRepository(testDataDir, nil)
	if err != nil {
		t.Error(err)
	}

	topupToCreate := domain.NewTopup("aa", 0, 1.54, time.Now())

	err = repo.AddTopup(context.Background(), topupToCreate)
	if err != nil {
		t.Error(err)
	}

	fakeAssetAmount := uint64(100)
	fakePset := "fakepartialbase64"
	topup, err := repo.SetCompleted(context.Background(), topupToCreate.ID, fakePset, fakeAssetAmount)
	assert.Nil(t, err)

	topups, err := repo.GetAllTopups(context.Background())
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, domain.Completed, topup.Status)
	assert.Equal(t, false, topup.CompletedAt.IsZero())
	assert.Equal(t, fakeAssetAmount, topup.AssetAmount)
	assert.Equal(t, fakePset, topup.Partial)

	assert.Equal(t, domain.Completed, topups[0].Status)
	assert.Equal(t, false, topups[0].CompletedAt.IsZero())
	assert.Equal(t, fakeAssetAmount, topups[0].AssetAmount)
	assert.Equal(t, fakePset, topups[0].Partial)
}

func TestSetSettled(t *testing.T) {
	dirManagementForTest(t)

	repo, err := NewBadgerTopupRepository(testDataDir, nil)
	if err != nil {
		t.Error(err)
	}

	topupToCreate := domain.NewTopup("aa", 0, 1.54, time.Now())

	err = repo.AddTopup(context.Background(), topupToCreate)
	if err != nil {
		t.Error(err)
	}

	topup, err := repo.SetSettled(context.Background(), topupToCreate.ID, "useless")
	assert.Nil(t, err)

	topups, err := repo.GetAllTopups(context.Background())
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, domain.Settled, topup.Status)
	assert.Equal(t, false, topup.SettledAt.IsZero())

	assert.Equal(t, domain.Settled, topups[0].Status)
	assert.Equal(t, false, topups[0].SettledAt.IsZero())
}

func dirManagementForTest(t *testing.T) {
	err := CreateDirIfNotExist(testDataDir)
	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		err := os.RemoveAll(testDataDir)
		if err != nil {
			t.Error(err)
		}
	})
}
