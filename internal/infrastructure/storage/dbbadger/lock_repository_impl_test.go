package dbbadger

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

const (
	txID = "4619ac256b1fa678ac0834c3f7908c6a589cf63580260e86eb11e67f9cb7b8c9"
)

func TestLock(t *testing.T) {
	clean, err := createTestDataDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(clean)

	repo, err := NewBadgerLockRepository(testDataDir, nil)
	if err != nil {
		t.Error(err)
	}

	lock := domain.Lock{
		TxID: txID,
		Vout: 1,
	}

	err = repo.Lock(context.Background(), []domain.Lock{lock})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, repo.IsLocked(context.Background(), lock))
}

func TestIsLocked(t *testing.T) {
	clean, err := createTestDataDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(clean)

	repo, err := NewBadgerLockRepository(testDataDir, nil)
	if err != nil {
		t.Error(err)
	}

	lock := domain.Lock{
		TxID: txID,
		Vout: 1,
	}

	lockNotLocked := domain.Lock{
		TxID: txID,
		Vout: 8,
	}

	err = repo.Lock(context.Background(), []domain.Lock{lock})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, repo.IsLocked(context.Background(), lock))
	assert.Equal(t, false, repo.IsLocked(context.Background(), lockNotLocked))
}

func TestUnlock(t *testing.T) {
	clean, err := createTestDataDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(clean)

	repo, err := NewBadgerLockRepository(testDataDir, nil)
	if err != nil {
		t.Error(err)
	}

	lock := domain.Lock{
		TxID: txID,
		Vout: 1,
	}

	err = repo.Lock(context.Background(), []domain.Lock{lock})
	if err != nil {
		t.Fatal(err)
	}

	if !repo.IsLocked(context.Background(), lock) {
		t.Fatal(errors.New("lock should be locked"))
	}

	err = repo.Unlock(context.Background(), lock)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, false, repo.IsLocked(context.Background(), lock))
}

func TestGetLocks(t *testing.T) {
	clean, err := createTestDataDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(clean)

	repo, err := NewBadgerLockRepository(testDataDir, nil)
	if err != nil {
		t.Error(err)
	}

	lock := domain.Lock{
		TxID: txID,
		Vout: 1,
	}

	err = repo.Lock(context.Background(), []domain.Lock{lock})
	if err != nil {
		t.Fatal(err)
	}

	locks, err := repo.GetLocks(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(locks))
}
