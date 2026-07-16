package index

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

// Load returns all items, refreshing the index cache from tickets and git
// history as needed.
func Load(store *storage.FS, repo gitx.Repo, opts Options) ([]*datamodel.Item, Result, error) {
	return loadRetry(store, repo, opts, false)
}

// Refresh reindexes tickets and git history without returning the loaded
// items, forcing a full reindex when full is true.
func Refresh(store *storage.FS, repo gitx.Repo, opts Options, full bool) (Result, error) {
	_, res, err := loadRetry(store, repo, opts, full)
	return res, err
}

func loadRetry(store *storage.FS, repo gitx.Repo, opts Options, force bool) ([]*datamodel.Item, Result, error) {
	items, res, err := loadOnce(store, repo, opts, force)
	if err == nil {
		return items, res, nil
	}
	if gitx.IsCmdError(err) {
		return nil, Result{}, err
	}
	if err := discard(store.CacheDir()); err != nil {
		return nil, Result{}, err
	}
	return loadOnce(store, repo, opts, true)
}

func loadOnce(store *storage.FS, repo gitx.Repo, opts Options, force bool) ([]*datamodel.Item, Result, error) {
	idx, err := Open(store.CacheDir())
	if err != nil {
		return nil, Result{}, err
	}
	defer func() { _ = idx.Close() }()
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
