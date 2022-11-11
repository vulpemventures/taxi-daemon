package db_test

import (
	"encoding/hex"
	postgresdb "github.com/vulpemventures/taxi-daemon/internal/infrastructure/storage/postgres"
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
	testAddGetScripts(t, repo)
}

func testAddGetScripts(t *testing.T, repo domain.WhitelistRepository) {
	whitelist, err := repo.GetScripts(ctx)
	require.NoError(t, err)
	require.Empty(t, whitelist)

	scripts := randomScripts()
	count, err := repo.AddScripts(ctx, scripts)
	require.NoError(t, err)
	require.Equal(t, len(scripts), count)

	whitelist, err = repo.GetScripts(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, whitelist)
	require.Len(t, whitelist, len(scripts))
	require.ElementsMatch(t, whitelist, scripts)

	newAddresses := randomScripts()
	scripts = append(scripts, newAddresses...)
	count, err = repo.AddScripts(ctx, scripts)
	require.NoError(t, err)
	require.Equal(t, len(newAddresses), count)

	whitelist, err = repo.GetScripts(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, whitelist)
	require.Len(t, whitelist, len(scripts))
	require.ElementsMatch(t, whitelist, scripts)
}

func newWhitelistRepositories() (map[string]domain.WhitelistRepository, error) {
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

	return map[string]domain.WhitelistRepository{
		"inmemory": inmemoryRepoManager.WhitelistRepository(),
		"badger":   badgerRepoManager.WhitelistRepository(),
		"postgres": pgRepoManager.WhitelistRepository(),
	}, nil
}

func randomScripts() []string {
	numOfScripts := randomIntInRange(2, 6)
	scripts := make([]string, 0, numOfScripts)
	for i := 0; i < numOfScripts; i++ {
		scripts = append(scripts, hex.EncodeToString(randomBytes(30)))
	}
	return scripts
}
