package postgresdb

import (
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
)

const (
	postgresDriver             = "postgres"
	insecureDataSourceTemplate = "postgresql://%s:%s@%s:%d/%s?sslmode=disable"
)

type pgRepoManager struct {
	db *sql.DB

	priceRepository domain.PriceRepository
	topupRepository domain.TopupRepository
	whitelistRepo   domain.WhitelistRepository
}

type DbConfig struct {
	DbUser             string
	DbPassword         string
	DbHost             string
	DbPort             int
	DbName             string
	MigrationSourceURL string
}

func NewRepoManager(dbConfig DbConfig) (ports.RepoManager, error) {
	db, err := connect(dbConfig)
	if err != nil {
		return nil, err
	}

	if err = migrateDb(db, dbConfig.MigrationSourceURL); err != nil {
		return nil, err
	}

	priceRepositoryImpl := NewPriceRepositoryImpl(db)
	topupRepositoryImpl := NewTopupRepositoryImpl(db)
	whitelistRepositoryImpl := NewWhitelistRepositoryImpl(db)

	return &pgRepoManager{
		db:              db,
		priceRepository: priceRepositoryImpl,
		topupRepository: topupRepositoryImpl,
		whitelistRepo:   whitelistRepositoryImpl,
	}, nil
}

func (p *pgRepoManager) TopupRepository() domain.TopupRepository {
	return p.topupRepository
}

func (p *pgRepoManager) PriceRepository() domain.PriceRepository {
	return p.priceRepository
}

func (p *pgRepoManager) WhitelistRepository() domain.WhitelistRepository {
	return p.whitelistRepo
}

func (p *pgRepoManager) Close() {
	p.db.Close()
}

func connect(dbConfig DbConfig) (*sql.DB, error) {
	dataSource := insecureDataSourceStr(dbConfig)

	db, err := sql.Open(
		postgresDriver,
		dataSource,
	)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func migrateDb(db *sql.DB, migrationSourceUrl string) error {
	dbInstance, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationSourceUrl,
		postgresDriver,
		dbInstance,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

// insecureDataSourceStr converts database configuration params to connection string
func insecureDataSourceStr(dbConfig DbConfig) string {
	return fmt.Sprintf(
		insecureDataSourceTemplate,
		dbConfig.DbUser,
		dbConfig.DbPassword,
		dbConfig.DbHost,
		dbConfig.DbPort,
		dbConfig.DbName,
	)
}
