package dbbadger

import (
	"os"
	"testing"
)

const (
	testDataDir = "./test-datadir"
)

func dirManagementForTest(t *testing.T) {
	err := CreateDirIfNotExist(testDataDir)
	if err != nil {
		t.Error(err)
	}

	t.Cleanup(func() {
		err := os.RemoveAll(testDataDir)
		if err != nil {
			t.Error(err)
		}
	})
}
