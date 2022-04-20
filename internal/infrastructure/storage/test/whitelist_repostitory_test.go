package db_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	dbbadger "github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/badger"
	"github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/inmemory"
)

func TestWhitelistRepository(t *testing.T) {
	repositories, err := newWhitelistRepositories()
	require.NoError(t, err)

	for name, repo := range repositories {
		t.Run(name, func(t *testing.T) {
			testWhitelistRepository(t, repo)
		})
	}
}

func testWhitelistRepository(t *testing.T, repo domain.WhitelistRepository) {
	testAddGetAddresses(t, repo)
}

func testAddGetAddresses(t *testing.T, repo domain.WhitelistRepository) {
	whitelist, err := repo.GetAddresses(ctx)
	require.NoError(t, err)
	require.Empty(t, whitelist)

	addresses := randomAddresses()
	count, err := repo.AddAddresses(ctx, addresses)
	require.NoError(t, err)
	require.Equal(t, len(addresses), count)

	whitelist, err = repo.GetAddresses(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, whitelist)
	require.Len(t, whitelist, len(addresses))
	require.ElementsMatch(t, whitelist, addresses)

	newAddresses := randomAddresses()
	addresses = append(addresses, newAddresses...)
	count, err = repo.AddAddresses(ctx, addresses)
	require.NoError(t, err)
	require.Equal(t, len(newAddresses), count)

	whitelist, err = repo.GetAddresses(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, whitelist)
	require.Len(t, whitelist, len(addresses))
	require.ElementsMatch(t, whitelist, addresses)
}

func newWhitelistRepositories() (map[string]domain.WhitelistRepository, error) {
	inmemoryRepoManager := inmemory.NewRepoManager()
	badgerRepoManager, err := dbbadger.NewRepoManager("", nil)
	if err != nil {
		return nil, err
	}

	return map[string]domain.WhitelistRepository{
		"inmemory": inmemoryRepoManager.WhitelistRepository(),
		"badger":   badgerRepoManager.WhitelistRepository(),
	}, nil
}

func randomAddresses() []string {
	numOfAddresses := randomIntInRange(2, 6)
	addresses := make([]string, 0, numOfAddresses)
	for i := 0; i < numOfAddresses; i++ {
		addresses = append(addresses, hex.EncodeToString(randomBytes(30)))
	}
	return addresses
}
