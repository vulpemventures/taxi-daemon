package dbbadger

import (
	"math/rand"
	"os"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	log "github.com/sirupsen/logrus"
	"github.com/timshannon/badgerhold/v2"
)

func createDb(dbDir string, logger badger.Logger) (*badgerhold.Store, error) {
	opts := badger.DefaultOptions(dbDir)
	opts.Compression = options.ZSTD
	opts.Logger = logger

	db, err := badgerhold.Open(badgerhold.Options{
		Encoder:          badgerhold.DefaultEncode,
		Decoder:          badgerhold.DefaultDecode,
		SequenceBandwith: 100,
		Options:          opts,
	})

	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		log.Info("set up badger garbage collection")

		for range ticker.C {
			log.Info("run garbage collector")
			err := db.Badger().RunValueLogGC(0.5)
			if err != nil && err != badger.ErrNoRewrite {
				log.Error(err)
			}
		}
	}()

	return db, nil
}

// CreateDirIfNotExist takes a path as parameter and test if it exist as directory (or create it if not)
func CreateDirIfNotExist(path string) error {
	info, err := os.Stat(path)

	if err == nil {
		if info.IsDir() {
			return nil
		}

		return ErrIsNotDirectory{Path: path}
	}

	if os.IsNotExist(err) {
		return os.Mkdir(path, os.ModeDir|0755)
	}

	return err
}

// isTransactionConflict returns wheter the error occured when commiting a
// transacton is a conflict
func isTransactionConflict(err error) bool {
	return err == badger.ErrConflict
}

func sleepSomeMS() {
	rand.Seed(time.Now().UnixNano())
	ms := time.Duration(rand.Intn(30) + 1)
	time.Sleep(ms * time.Millisecond)
}

func runTransaction(handler func() (interface{}, error)) (interface{}, error) {
	for {
		res, err := handler()
		if err != nil {
			if isTransactionConflict(err) {
				sleepSomeMS()
				continue
			}

			return nil, err
		}
		return res, nil
	}
}
