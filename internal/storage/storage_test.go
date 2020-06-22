package storage_test

import (
	"testing"

	"github.com/vulpemventures/liquid-taxi/internal/storage"
	"github.com/vulpemventures/liquid-taxi/pkg/fees"
)

func TestNewStorageFail(t *testing.T) {
	_, err := storage.NewStorage("mongodb+srv://foobarBadUrl")
	if err == nil {
		t.Errorf("Should have failed")
		return
	}
}

func TestStorage_InsertOrder(t *testing.T) {
	s, err := storage.NewStorage("mongodb://127.0.0.1:27017/test")
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	objectID, err := s.InsertOrder(2, `foo`, `bar`, &fees.Breakdown{Size: 200})
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	got, _, err := s.GetOrderByID(objectID)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if got.Unsigned != "foo" || got.Partial != "bar" {
		t.Errorf("Inserted values do not match")
	}

	_, notFound, err := s.GetOrderByID("5ef090999ca3c69e5c836de8")
	if !notFound && err != nil {
		t.Errorf(err.Error())
		return
	}
	if !notFound {
		t.Errorf("Should have thown")
		return
	}

	s.Close()

}
