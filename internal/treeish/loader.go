// Package treeish loads a kira item set and its config from any git tree-ish
// in one batched cat-file pass, backing read-only time-travel views and diffs.
package treeish

import (
	"fmt"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/storage"
)

type Loaded struct {
	Treeish  string
	Items    []*datamodel.Item
	Config   *datamodel.Config
	Snapshot id.Snapshot
	Resolver *id.Resolver
	Warnings []string
}

func Load(repo gitx.Repo, treeish string) (*Loaded, error) {
	names, err := repo.LsTreeNames(treeish, storage.TicketsPrefix, storage.ConfigRelPath)
	if err != nil {
		return nil, err
	}
	var ticketPaths []string
	hasConfig := false
	for _, n := range names {
		switch {
		case n == storage.ConfigRelPath:
			hasConfig = true
		case storage.IsItemPath(n):
			ticketPaths = append(ticketPaths, n)
		}
	}
	if !hasConfig {
		return nil, errNoConfigAt(treeish)
	}

	specs := make([]string, 0, len(ticketPaths)+1)
	specs = append(specs, gitx.RevPath(treeish, storage.ConfigRelPath))
	for _, p := range ticketPaths {
		specs = append(specs, gitx.RevPath(treeish, p))
	}
	blobs, err := repo.CatFileBatch(specs)
	if err != nil {
		return nil, err
	}

	if !blobs[0].Found {
		return nil, errNoConfigAt(treeish)
	}
	cfg, err := config.Parse([]byte(blobs[0].Content))
	if err != nil {
		return nil, fmt.Errorf("config at %s: %w", treeish, err)
	}

	items := make([]*datamodel.Item, 0, len(ticketPaths))
	var warnings []string
	for i, p := range ticketPaths {
		blob := blobs[i+1]
		if !blob.Found {
			return nil, fmt.Errorf("%s listed at %s but blob is missing (corrupt tree)", p, treeish)
		}
		it, err := codec.Parse(blob.Content)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped %s at %s: %v", p, treeish, err))
			continue
		}
		it.Activity = it.Updated
		items = append(items, it)
	}

	snap := storage.Snapshot(cfg.Project.Key, items)
	return &Loaded{
		Treeish:  treeish,
		Items:    items,
		Config:   cfg,
		Snapshot: snap,
		Resolver: id.NewResolver(snap),
		Warnings: warnings,
	}, nil
}

func errNoConfigAt(treeish string) error {
	return fmt.Errorf("no %s at %s (cannot time-travel before kira init)", storage.ConfigRelPath, treeish)
}
