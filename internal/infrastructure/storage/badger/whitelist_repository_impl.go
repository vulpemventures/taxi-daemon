package dbbadger

import (
	"context"

	badgerhold "github.com/timshannon/badgerhold/v4"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
)

type whitelistRepository struct {
	store *badgerhold.Store
}

func NewWhitelistRepository(store *badgerhold.Store) domain.WhitelistRepository {
	return newWhitelistRepository(store)
}

func newWhitelistRepository(store *badgerhold.Store) *whitelistRepository {
	return &whitelistRepository{store}
}

func (r *whitelistRepository) AddAddresses(
	ctx context.Context, addresses []string,
) (int, error) {
	count := 0
	for _, addr := range addresses {
		done, err := r.insertAddress(ctx, addr)
		if err != nil {
			return -1, err
		}
		if done {
			count++
		}
	}
	return count, nil
}

func (r *whitelistRepository) GetAddresses(
	_ context.Context,
) ([]string, error) {
	whitelist := make([]address, 0)
	if err := r.store.Find(&whitelist, nil); err != nil {
		return nil, err
	}

	addresses := make([]string, 0, len(whitelist))
	for _, w := range whitelist {
		addresses = append(addresses, w.Addr)
	}
	return addresses, nil
}

func (r *whitelistRepository) close() {
	r.store.Close()
}

func (r *whitelistRepository) insertAddress(
	_ context.Context, addr string,
) (bool, error) {
	item := address{addr}
	if err := r.store.Insert(addr, item); err != nil {
		if err == badgerhold.ErrKeyExists {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

type address struct {
	Addr string
}
