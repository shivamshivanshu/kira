package index

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func Load(store *storage.Store, repo gitx.Repo, opts Options) ([]*datamodel.Item, Result, error) {
	return loadRetry(store, repo, opts, false)
}

func Refresh(store *storage.Store, repo gitx.Repo, opts Options, full bool) (Result, error) {
	_, res, err := loadRetry(store, repo, opts, full)
	return res, err
}

func loadRetry(store *storage.Store, repo gitx.Repo, opts Options, force bool) ([]*datamodel.Item, Result, error) {
	items, res, err := loadOnce(store, repo, opts, force)
	if err == nil {
		return items, res, nil
	}
	discard(store.CacheDir())
	return loadOnce(store, repo, opts, true)
}

func loadOnce(store *storage.Store, repo gitx.Repo, opts Options, force bool) ([]*datamodel.Item, Result, error) {
	idx, err := Open(store.CacheDir())
	if err != nil {
		return nil, Result{}, err
	}
	defer idx.Close()
	res, err := idx.reindex(store, repo, opts, force)
	if err != nil {
		return nil, Result{}, err
	}
	items, err := idx.Items()
	if err != nil {
		return nil, Result{}, err
	}
	return items, res, nil
}
