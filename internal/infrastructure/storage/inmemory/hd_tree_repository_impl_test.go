package inmemory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/go-elements/network"
)

const (
	mnemonic = "clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"
)

func TestGetOrCreateHDTree(t *testing.T) {
	repo := newInmemoryHDTreeRepository(words, network.Regtest)
	ctx := context.Background()

	assert.Nil(t, repo.hdTree)
	if _, err := repo.GetOrCreateHDTree(ctx); err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, repo.hdTree)
}

func TestGetNewFundingAddresses(t *testing.T) {
	repo := newInmemoryHDTreeRepository(words, network.Regtest)
	ctx := context.Background()

	if _, err := repo.GetOrCreateHDTree(ctx); err != nil {
		t.Fatal(err)
	}

	addresses, err := repo.GetNewFundingAddresses(ctx, 3, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 3, len(addresses))
}

func TestGetNewPayoutAddresses(t *testing.T) {
	repo := newInmemoryHDTreeRepository(words, network.Regtest)
	ctx := context.Background()

	if _, err := repo.GetOrCreateHDTree(ctx); err != nil {
		t.Fatal(err)
	}

	addresses, err := repo.GetNewPayoutAddresses(ctx, 3, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 3, len(addresses))
}
