package dbbadger

import (
	"os"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
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

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
		again:
			err := db.Badger().RunValueLogGC(0.5)
			if err == nil {
				goto again
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
