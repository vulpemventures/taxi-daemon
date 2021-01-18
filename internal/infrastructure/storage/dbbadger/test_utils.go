package dbbadger

import (
	"os"
	"time"
)

const (
	testDataDir = "./test-datadir"
	utxoTTL     = 5 * time.Minute
)

func createTestDataDir() (func(), error) {
	err := CreateDirIfNotExist(testDataDir)
	if err != nil {
		return nil, err
	}

	cleanFn := func() {
		os.RemoveAll(testDataDir)
	}

	return cleanFn, nil
}
