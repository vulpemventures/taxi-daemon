package dbbadger

import (
	"math/rand"
	"os"
	"time"

	"github.com/dgraph-io/badger/v2"
	log "github.com/sirupsen/logrus"
	"github.com/timshannon/badgerhold/v2"
)

func createDb(dbDir string, logger badger.Logger) (*badgerhold.Store, error) {
	opts := badger.DefaultOptions(dbDir)
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

	ticker := time.NewTicker(30 * time.Minute)

	go func() {
		time.Sleep(1 * time.Minute)
		log.Debug("set up badger garbage collection for ", dbDir)

		for range ticker.C {
			log.Debug("run garbage collector on ", dbDir)
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

func encodeKey(key interface{}, typeName string) ([]byte, error) {
	const prefix = "bh_"
	encoded, err := badgerhold.DefaultEncode(key)
	if err != nil {
		return nil, err
	}

	return append([]byte(prefix+typeName), encoded...), nil
}
