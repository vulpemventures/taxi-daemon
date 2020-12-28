package inmemory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/liquid-taxi/config"
)

const (
	mnemonic = "clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"
)

func TestGetOrCreateHDTree(t *testing.T) {
	repo := newInmemoryHDTreeRepository()
	ctx := context.Background()

	assert.Nil(t, repo.hdTree)
	if _, err := repo.GetOrCreateHDTree(ctx); err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, repo.hdTree)
}

func TestGetNewFundingAddresses(t *testing.T) {
	repo := newInmemoryHDTreeRepository()
	ctx := context.Background()

	config.Set(config.MnemonicKey, mnemonic)
	t.Cleanup(func() {
		config.Set(config.MnemonicKey, "")
	})

	if _, err := repo.GetOrCreateHDTree(ctx); err != nil {
		t.Fatal(err)
	}

	addresses, err := repo.GetNewFundingAddresses(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 3, len(addresses))
}

func TestGetNewPayoutAddresses(t *testing.T) {
	repo := newInmemoryHDTreeRepository()
	ctx := context.Background()

	config.Set(config.MnemonicKey, mnemonic)
	t.Cleanup(func() {
		config.Set(config.MnemonicKey, "")
	})

	if _, err := repo.GetOrCreateHDTree(ctx); err != nil {
		t.Fatal(err)
	}

	addresses, err := repo.GetNewPayoutAddresses(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 3, len(addresses))
}
