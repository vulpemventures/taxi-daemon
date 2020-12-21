package inmemory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tj/assert"
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

	_, hasBeenAddedToRepoMap := repo.utxosKeys[lock]

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

	if !repo.IsLocked(context.Background(), lock) {
		t.Fatal(errors.New("lock should be locked"))
	}

	err = repo.Unlock(context.Background(), lock)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, false, repo.IsLocked(context.Background(), lock))
}

func TestConcurrency(t *testing.T) {
	repo := newInmemoryLockRepository()

	lock := domain.Lock{
		TxID: txID,
		Vout: 1,
	}

	lockBis := domain.Lock{
		TxID: txID,
		Vout: 2,
	}

	emptyCtx := context.Background()

	go func() {
		err := repo.Lock(emptyCtx, []domain.Lock{lock, lockBis})
		assert.Nil(t, err)
		time.Sleep(2 * time.Second)
		err = repo.Unlock(emptyCtx, lockBis)
		assert.Nil(t, err)
		time.Sleep(1 * time.Second)
		err = repo.Lock(emptyCtx, []domain.Lock{lock})
		assert.Nil(t, err)
	}()

	go func() {
		err := repo.Lock(emptyCtx, []domain.Lock{lock})
		assert.Nil(t, err)
		time.Sleep(1 * time.Second)
		repo.IsLocked(emptyCtx, lockBis)
		err = repo.Unlock(emptyCtx, lock)
		assert.Nil(t, err)
	}()

}
