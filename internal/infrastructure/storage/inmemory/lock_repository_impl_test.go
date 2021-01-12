package inmemory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

const (
	txID = "4619ac256b1fa678ac0834c3f7908c6a589cf63580260e86eb11e67f9cb7b8c9"
)

func TestLock(t *testing.T) {
	repo := newInmemoryLockRepository()

	lock := domain.Lock{
		TxID: txID,
		Vout: 1,
	}

	err := repo.Lock(context.Background(), []domain.Lock{lock})
	if err != nil {
		t.Fatal(err)
	}

	_, hasBeenAddedToRepoMap := repo.locks[lock]

	assert.Equal(t, true, hasBeenAddedToRepoMap)
}

func TestIsLocked(t *testing.T) {
	repo := newInmemoryLockRepository()

	lock := domain.Lock{
		TxID: txID,
		Vout: 1,
	}

	err := repo.Lock(context.Background(), []domain.Lock{lock})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, repo.IsLocked(context.Background(), lock))
}

func TestUnlock(t *testing.T) {
	repo := newInmemoryLockRepository()

	lock := domain.Lock{
		TxID: txID,
		Vout: 1,
	}

	err := repo.Lock(context.Background(), []domain.Lock{lock})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, repo.IsLocked(context.Background(), lock))

	err = repo.Unlock(context.Background(), lock)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, false, repo.IsLocked(context.Background(), lock))
}

func TestGetLocks(t *testing.T) {
	repo := newInmemoryLockRepository()

	lock := domain.Lock{
		TxID: txID,
		Vout: 1,
	}

	err := repo.Lock(context.Background(), []domain.Lock{lock})
	if err != nil {
		t.Fatal(err)
	}

	locks, err := repo.GetLocks(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(locks))
}
