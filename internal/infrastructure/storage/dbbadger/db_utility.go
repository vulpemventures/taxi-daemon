package dbbadger

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/dgraph-io/badger/v2"
	"github.com/timshannon/badgerhold/v2"
)

func createDb(dbDir string, logger badger.Logger) (db *badgerhold.Store, err error) {
	opts := badger.DefaultOptions(dbDir)
	opts.Logger = logger

	db, err = badgerhold.Open(badgerhold.Options{
		Encoder:          JSONEncode,
		Decoder:          JSONDecode,
		SequenceBandwith: 100,
		Options:          opts,
	})

	return db, err
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

// JSONEncode is a custom JSON based encoder for badger
func JSONEncode(value interface{}) ([]byte, error) {
	var buff bytes.Buffer

	en := json.NewEncoder(&buff)

	err := en.Encode(value)
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

// JSONDecode is a custom JSON based decoder for badger
func JSONDecode(data []byte, value interface{}) error {
	var buff bytes.Buffer
	de := json.NewDecoder(&buff)

	_, err := buff.Write(data)
	if err != nil {
		return err
	}

	return de.Decode(value)
}
