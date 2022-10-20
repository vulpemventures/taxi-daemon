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

func (r *whitelistRepository) AddScripts(
	_ context.Context, scripts []string,
) (int, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	count := 0
	for _, script := range scripts {
		if _, ok := r.whitelist[script]; !ok {
			r.whitelist[script] = struct{}{}
			count++
		}
	}
	return count, nil
}

func (r *whitelistRepository) GetScripts(_ context.Context) ([]string, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	scripts := make([]string, 0, len(r.whitelist))
	for script := range r.whitelist {
		scripts = append(scripts, script)
	}
	return scripts, nil
}

func (r *whitelistRepository) RemoveScripts(
	_ context.Context, scripts []string,
) (int, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	count := 0
	for _, script := range scripts {
		if _, ok := r.whitelist[script]; ok {
			delete(r.whitelist, script)
			count++
		}
	}
	return count, nil
}
