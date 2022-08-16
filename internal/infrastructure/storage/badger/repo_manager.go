package dbbadger

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/options"
	log "github.com/sirupsen/logrus"
	badgerhold "github.com/timshannon/badgerhold/v4"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
)

type repoManager struct {
	priceRepository     *priceRepository
	topupRepository     *topupRepository
	whitelistRepository *whitelistRepository
}

func NewRepoManager(
	baseDbDir string, logger badger.Logger,
) (ports.RepoManager, error) {
	var priceDbDir, topupDbDir, whitelistDbDir string
	if len(baseDbDir) > 0 {
		priceDbDir = filepath.Join(baseDbDir, "price")
		topupDbDir = filepath.Join(baseDbDir, "topup")
		whitelistDbDir = filepath.Join(baseDbDir, "whitelist")
	}

	priceDb, err := createDb(priceDbDir, logger)
	if err != nil {
		return nil, fmt.Errorf("error while opening price db: %s", err)
	}
	topupDb, err := createDb(topupDbDir, logger)
	if err != nil {
		return nil, fmt.Errorf("error while opening topup db: %s", err)
	}
	whitelistDb, err := createDb(whitelistDbDir, logger)
	if err != nil {
		return nil, fmt.Errorf("error while opening whitelist db: %s", err)
	}

	priceRepo := newPriceRepository(priceDb)
	topupRepo := newTopupRepository(topupDb)
	whitelistRepo := newWhitelistRepository(whitelistDb)

	return &repoManager{
		priceRepository:     priceRepo,
		topupRepository:     topupRepo,
		whitelistRepository: whitelistRepo,
	}, nil
}

func (rm *repoManager) PriceRepository() domain.PriceRepository {
	return rm.priceRepository
}

func (rm *repoManager) TopupRepository() domain.TopupRepository {
	return rm.topupRepository
}

func (rm *repoManager) WhitelistRepository() domain.WhitelistRepository {
	return rm.whitelistRepository
}

func (rm *repoManager) Close() {
	rm.priceRepository.close()
	rm.topupRepository.close()
	rm.whitelistRepository.close()
}

func createDb(dbDir string, logger badger.Logger) (*badgerhold.Store, error) {
	isInMemory := len(dbDir) <= 0

	opts := badger.DefaultOptions(dbDir)
	opts.Logger = logger

	if isInMemory {
		opts.InMemory = true
	} else {
		opts.Compression = options.ZSTD
	}

	db, err := badgerhold.Open(badgerhold.Options{
		Encoder:          badgerhold.DefaultEncode,
		Decoder:          badgerhold.DefaultDecode,
		SequenceBandwith: 100,
		Options:          opts,
	})
	if err != nil {
		return nil, err
	}

	if !isInMemory {
		ticker := time.NewTicker(30 * time.Minute)

		go func() {
			for {
				<-ticker.C
				if err := db.Badger().RunValueLogGC(0.5); err != nil && err != badger.ErrNoRewrite {
					log.Warnf("garbage collector: %s", err)
				}
			}
		}()
	}

	return db, nil
}
