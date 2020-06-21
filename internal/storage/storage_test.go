package storage_test

import (
	"testing"

	"github.com/vulpemventures/liquid-taxi/internal/storage"
	"github.com/vulpemventures/liquid-taxi/pkg/fees"
)

func TestStorage_InsertOrder(t *testing.T) {

	s, err := storage.NewStorage("")
	if err != nil {
		//t.Errorf(err.Error())
		return
	}
	_, err = s.InsertOrder(2, `foo`, `bar`, fees.Breakdown{Size: 100})
	if err != nil {
		t.Errorf("Storage.InsertOrder() error")
		return
	}

}
