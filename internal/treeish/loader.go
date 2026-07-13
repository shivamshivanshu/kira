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

const (
	ticketsPrefix = ".kira/tickets"
	configPath    = ".kira/config.yaml"
)

type Loaded struct {
	Treeish  string
	Items    []*datamodel.Item
	Config   *datamodel.Config
	Snapshot id.Snapshot
	Resolver *id.Resolver
}

func Load(repo gitx.Repo, treeish string) (*Loaded, error) {
	names, err := repo.LsTreeNames(treeish, ticketsPrefix, configPath)
	if err != nil {
		return nil, err
	}
	var ticketPaths []string
	hasConfig := false
	for _, n := range names {
		switch {
		case n == configPath:
			hasConfig = true
		case storage.IsItemPath(n):
			ticketPaths = append(ticketPaths, n)
		}
	}
	if !hasConfig {
		return nil, fmt.Errorf("no %s at %s (cannot time-travel before kira init)", configPath, treeish)
	}

	specs := make([]string, 0, len(ticketPaths)+1)
	specs = append(specs, treeish+":"+configPath)
	for _, p := range ticketPaths {
		specs = append(specs, treeish+":"+p)
	}
	blobs, err := repo.CatFileBatch(specs)
	if err != nil {
		return nil, err
	}

	if !blobs[0].Found {
		return nil, fmt.Errorf("no %s at %s (cannot time-travel before kira init)", configPath, treeish)
	}
	cfg, err := config.Parse([]byte(blobs[0].Content))
	if err != nil {
		return nil, fmt.Errorf("config at %s: %w", treeish, err)
	}

	items := make([]*datamodel.Item, 0, len(ticketPaths))
	for i, p := range ticketPaths {
		blob := blobs[i+1]
		if !blob.Found {
			return nil, fmt.Errorf("%s listed at %s but blob is missing (corrupt tree)", p, treeish)
		}
		it, err := codec.Parse(blob.Content)
		if err != nil {
			return nil, fmt.Errorf("parsing %s at %s: %w", p, treeish, err)
		}
		items = append(items, it)
	}

	snap := storage.Snapshot(cfg.Project.Key, items)
	return &Loaded{
		Treeish:  treeish,
		Items:    items,
		Config:   cfg,
		Snapshot: snap,
		Resolver: id.NewResolver(snap),
	}, nil
}
