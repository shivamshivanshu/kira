package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func snapshotAndResolver(key string, items []*datamodel.Item) (id.Snapshot, *id.Resolver) {
	snap := storage.Snapshot(key, items)
	return snap, id.NewResolver(snap)
}

func indexOptions(cfg *datamodel.Config) index.Options {
	return index.Options{
		ProjectKey:       cfg.Project.Key,
		BoardKeys:        cfg.BoardKeys(),
		TrailerKey:       cfg.Commit.Trailer,
		CloseTrailer:     cfg.Commit.CloseTrailer,
		SubjectPrefix:    cfg.Commit.SubjectPrefix,
		LandedRef:        cfg.Git.LandedRef,
		LinkMarkers:      cfg.Commit.LinkMarkers,
		ReferenceMarkers: cfg.Commit.ReferenceMarkers,
	}
}

func (s *Store) CachedItems() ([]*datamodel.Item, error) {
	return index.ReadCached(s.fs().CacheDir())
}

func (s *Store) Index(cfg *datamodel.Config, full, closes bool) (*datamodel.IndexResult, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	opts := indexOptions(cfg)
	if closes {
		opts.Closes = true
		opts.LandedRef = s.landedRef(cfg)
	}
	res, err := index.Refresh(s.fs(), s.repo(), opts, full)
	if err != nil {
		return nil, err
	}
	result := &datamodel.IndexResult{Action: res.Action, Reason: res.Reason, Items: res.Items, Closed: []string{}}
	if closes {
		closed, notes, err := s.applyCloses(cfg, res.Closes)
		if err != nil {
			return nil, err
		}
		result.Closed = nonNil(closed)
		result.StderrNotes = notes
	}
	return result, nil
}
