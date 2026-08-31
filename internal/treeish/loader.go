// Package treeish loads a kira item set, its config and its schemas from any
// git tree-ish in one batched cat-file pass, backing read-only time-travel
// views and diffs.
package treeish

import (
	"fmt"
	"slices"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/storage"
)

type Loaded struct {
	Items    []*datamodel.Item
	Config   *datamodel.Config
	Schemas  map[string]entityschema.Schema
	Resolver *id.Resolver
	Warnings []string
}

func Load(repo gitx.Repo, treeish string) (*Loaded, error) {
	sha, err := repo.ResolveTreeish(treeish)
	if err != nil {
		return nil, errx.User("resolving %s: %v", treeish, err)
	}

	names, err := repo.LsTreeNames(sha, storage.TicketsPrefix, storage.SchemaPrefix, storage.ConfigRelPath)
	if err != nil {
		return nil, err
	}
	var ticketPaths, schemaPaths []string
	hasConfig := false
	for _, n := range names {
		switch {
		case n == storage.ConfigRelPath:
			hasConfig = true
		case storage.IsItemPath(n):
			ticketPaths = append(ticketPaths, n)
		case storage.IsSchemaPath(n):
			schemaPaths = append(schemaPaths, n)
		}
	}
	if !hasConfig {
		return nil, errConfigMissing(sha)
	}

	specs := make([]string, 0, len(ticketPaths)+len(schemaPaths)+1)
	specs = append(specs, gitx.RevPath(sha, storage.ConfigRelPath))
	for _, p := range slices.Concat(ticketPaths, schemaPaths) {
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
	blobWarning := func(p string) string {
		return fmt.Sprintf("skipped %s at %s: listed in the tree but its blob is missing (corrupt tree)", p, sha)
	}
	for i, p := range ticketPaths {
		blob := blobs[i+1]
		if !blob.Found {
			warnings = append(warnings, blobWarning(p))
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

	var schemaFiles []entityschema.File
	for i, p := range schemaPaths {
		blob := blobs[len(ticketPaths)+i+1]
		if !blob.Found {
			warnings = append(warnings, blobWarning(p))
			continue
		}
		schemaFiles = append(schemaFiles, entityschema.File{Name: p, Data: []byte(blob.Content)})
	}
	schemas, err := entityschema.LoadFiles(schemaFiles)
	if err != nil {
		return nil, errx.User("schema at %s: %v", sha, err)
	}

	_, resolver := storage.SnapshotAndResolver(cfg.Project.Key, items)
	return &Loaded{
		Items:    items,
		Config:   cfg,
		Schemas:  schemas,
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
