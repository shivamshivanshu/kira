// Package treeish loads a kira item set and its config from any git tree-ish
// in one batched cat-file pass, backing read-only time-travel views and diffs.
package treeish

import (
	"fmt"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/storage"
)

type Loaded struct {
	Items    []*datamodel.Item
	Config   *datamodel.Config
	Resolver *id.Resolver
	Warnings []string
}

func Load(repo gitx.Repo, treeish string) (*Loaded, error) {
	sha, err := repo.ResolveTreeish(treeish)
	if err != nil {
		return nil, errx.User("resolving %s: %v", treeish, err)
	}

	names, err := repo.LsTreeNames(sha, storage.TicketsPrefix, storage.ConfigRelPath)
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
		return nil, errConfigMissing(sha)
	}

	specs := make([]string, 0, len(ticketPaths)+1)
	specs = append(specs, gitx.RevPath(sha, storage.ConfigRelPath))
	for _, p := range ticketPaths {
		specs = append(specs, gitx.RevPath(sha, p))
	}
	blobs, err := repo.CatFileBatch(specs)
	if err != nil {
		return nil, err
	}

	if !blobs[0].Found {
		return nil, errConfigCorrupt(sha)
	}
	// Deliberately config.Parse, not config.LoadWithUser: no user tier for
	// a historical blob, and nothing here reads the fields it would add.
	cfg, err := config.Parse([]byte(blobs[0].Content))
	if err != nil {
		return nil, errx.User("config at %s: %w", sha, err)
	}

	items := make([]*datamodel.Item, 0, len(ticketPaths))
	var warnings []string
	for i, p := range ticketPaths {
		blob := blobs[i+1]
		if !blob.Found {
			warnings = append(warnings, fmt.Sprintf("skipped %s at %s: listed in the tree but its blob is missing (corrupt tree)", p, sha))
			continue
		}
		it, err := codec.Parse(blob.Content)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skipped %s at %s: %v", p, sha, err))
			continue
		}
		it.Activity = it.Updated
		items = append(items, it)
	}

	_, resolver := storage.SnapshotAndResolver(cfg.Project.Key, items)
	return &Loaded{
		Items:    items,
		Config:   cfg,
		Resolver: resolver,
		Warnings: warnings,
	}, nil
}

func errConfigMissing(sha string) error {
	return errx.User("no %s at %s", storage.ConfigRelPath, sha).
		WithHint("cannot time-travel before kira init")
}

func errConfigCorrupt(sha string) error {
	return errx.User("%s is listed in the tree at %s but its blob is missing", storage.ConfigRelPath, sha).
		WithHint("the repository's object database may be corrupt")
}
