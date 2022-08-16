package inmemory

import (
	"context"
	"sync"

	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
)

type whitelistRepository struct {
	whitelist map[string]struct{}
	lock      *sync.RWMutex
}

// NewWhitelistRepository creates a new inmemory WhitelistRepository implementation
func NewWhitelistRepository() domain.WhitelistRepository {
	return newWhitelistRepository()
}

func newWhitelistRepository() *whitelistRepository {
	return &whitelistRepository{
		whitelist: make(map[string]struct{}),
		lock:      &sync.RWMutex{},
	}
}

func (r *whitelistRepository) AddAddresses(
	_ context.Context, addresses []string,
) (int, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	count := 0
	for _, addr := range addresses {
		if _, ok := r.whitelist[addr]; !ok {
			r.whitelist[addr] = struct{}{}
			count++
		}
	}
	return count, nil
}

func (r *whitelistRepository) GetAddresses(_ context.Context) ([]string, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	addresses := make([]string, 0, len(r.whitelist))
	for addr := range r.whitelist {
		addresses = append(addresses, addr)
	}
	return addresses, nil
}

func (r *whitelistRepository) RemoveAddresses(
	_ context.Context, addresses []string,
) (int, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	count := 0
	for _, addr := range addresses {
		if _, ok := r.whitelist[addr]; ok {
			delete(r.whitelist, addr)
			count++
		}
	}
	return count, nil
}
