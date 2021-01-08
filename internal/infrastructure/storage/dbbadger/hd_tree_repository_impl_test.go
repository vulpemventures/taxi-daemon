package dbbadger

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/go-elements/network"
)

const (
	mnemonic = "clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"
)

func TestGetOrCreateHDTree(t *testing.T) {
	clean, err := createTestDataDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(clean)

	repo, err := NewBadgerHDTreeRepository(testDataDir, strings.Split(mnemonic, " "), network.Regtest, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	hdTree, err := repo.GetOrCreateHDTree(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, hdTree)
}

func TestGetNewFundingAddresses(t *testing.T) {
	clean, err := createTestDataDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		clean()
	})

	repo, err := NewBadgerHDTreeRepository(testDataDir, strings.Split(mnemonic, " "), network.Regtest, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	addresses, err := repo.GetNewFundingAddresses(ctx, 3, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 3, len(addresses))
}

func TestGetNewPayoutAddresses(t *testing.T) {
	clean, err := createTestDataDir()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		clean()
	})

	repo, err := NewBadgerHDTreeRepository(testDataDir, strings.Split(mnemonic, " "), network.Regtest, nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	addresses, err := repo.GetNewPayoutAddresses(ctx, 3, true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 3, len(addresses))
}
