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

func (r *whitelistRepository) AddScripts(
	ctx context.Context, scripts []string,
) (int, error) {
	count := 0
	for _, script := range scripts {
		done, err := r.insertScript(ctx, script)
		if err != nil {
			return -1, err
		}
		if done {
			count++
		}
	}
	return count, nil
}

func (r *whitelistRepository) GetScripts(
	_ context.Context,
) ([]string, error) {
	whitelist := make([]scriptStruct, 0)
	if err := r.store.Find(&whitelist, nil); err != nil {
		return nil, err
	}

	scripts := make([]string, 0, len(whitelist))
	for _, w := range whitelist {
		scripts = append(scripts, w.Script)
	}
	return scripts, nil
}

func (r *whitelistRepository) RemoveScripts(
	ctx context.Context, scripts []string,
) (int, error) {
	count := 0
	for _, script := range scripts {
		done, err := r.deleteScript(ctx, script)
		if err != nil {
			return -1, err
		}
		if done {
			count++
		}
	}
	return count, nil
}

func (r *whitelistRepository) close() {
	r.store.Close()
}

func (r *whitelistRepository) insertScript(
	_ context.Context, script string,
) (bool, error) {
	item := scriptStruct{script}
	if err := r.store.Insert(script, item); err != nil {
		if err == badgerhold.ErrKeyExists {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (r *whitelistRepository) deleteScript(
	_ context.Context, script string,
) (bool, error) {
	item := scriptStruct{script}
	if err := r.store.Delete(script, item); err != nil {
		if err == badgerhold.ErrNotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

type scriptStruct struct {
	Script string
}
