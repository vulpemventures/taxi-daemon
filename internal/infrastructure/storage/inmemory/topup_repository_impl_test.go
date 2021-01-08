package inmemory

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

var (
	doesNotMatterPrice       = decimal.NewFromFloat(10.897)
	topup0                   = domain.NewTopup(assetHash, 5000, doesNotMatterPrice, time.Now())
	topup1                   = domain.NewTopup(assetHash, 5000, doesNotMatterPrice, time.Now())
	topupWithCompletedStatus = domain.Topup{
		ID:     "test",
		Status: domain.Completed,
	}
)

func TestAddTopup(t *testing.T) {
	repo := newInmemoryTopupRepository()
	err := repo.AddTopup(context.Background(), topup0)
	if err != nil {
		t.Fatal(err)
	}

	topup, ok := repo.topups[topup0.ID]
	assert.Equal(t, true, ok)
	assert.Equal(t, topup0, topup)
}

func TestGetAllTopups(t *testing.T) {
	repo := newInmemoryTopupRepository()
	err := repo.AddTopup(context.Background(), topup0)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.AddTopup(context.Background(), topup1)
	if err != nil {
		t.Fatal(err)
	}

	topups, err := repo.GetAllTopups(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(topups))
}

func TestMarkAsCompleted(t *testing.T) {
	repo := newInmemoryTopupRepository()
	err := repo.AddTopup(context.Background(), topup0)
	if err != nil {
		t.Fatal(err)
	}

	opts := domain.MarkAsCompletedOpts{
		TopupID:            topup0.ID,
		Partial:            "PSETbase64",
		BlindingPrivateKey: "fakeprivatekeyhex",
		BlindingPublicKey:  "fakeprivatekeyhex",
		AssetAmount:        100,
	}

	topupUpdated, err := repo.MarkAsCompleted(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, domain.Completed, topupUpdated.Status)

	// failing test
	err = repo.AddTopup(context.Background(), topupWithCompletedStatus)
	if err != nil {
		t.Fatal(err)
	}

	optsBis := opts
	optsBis.TopupID = topupWithCompletedStatus.ID

	_, err = repo.MarkAsCompleted(context.Background(), optsBis)

	assert.NotNil(t, err)
}

func TestMarkAsSettled(t *testing.T) {

	repo := newInmemoryTopupRepository()
	err := repo.AddTopup(context.Background(), topupWithCompletedStatus)
	if err != nil {
		t.Fatal(err)
	}

	topupUpdated, err := repo.MarkAsSettled(context.Background(), topupWithCompletedStatus.ID, "txID")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, domain.Settled, topupUpdated.Status)

	// failing test
	err = repo.AddTopup(context.Background(), topup0)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repo.MarkAsSettled(context.Background(), topup0.ID, "txID")

	assert.NotNil(t, err)
}
