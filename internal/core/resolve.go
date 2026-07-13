package core

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/merge"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func (s *Store) Resolve(refs []string, interactive bool) (*datamodel.ResolveResult, error) {
	repo := s.repo()
	unmerged, err := repo.UnmergedPaths()
	if err != nil {
		return nil, errx.User("%v", err)
	}

	var paths []string
	for _, p := range unmerged {
		if storage.IsItemPath(p) {
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return &datamodel.ResolveResult{}, nil
	}

	specs := make([]string, 0, len(paths)*3)
	for _, p := range paths {
		specs = append(specs, ":1:"+p, ":2:"+p, ":3:"+p)
	}
	blobs, err := repo.CatFileBatch(specs)
	if err != nil {
		return nil, errx.User("%v", err)
	}

	result := &datamodel.ResolveResult{}
	remote := remoteSide(repo)
	var staged []string
	for i, path := range paths {
		base, ours, theirs := blobs[3*i], blobs[3*i+1], blobs[3*i+2]
		if !ours.Found || !theirs.Found {
			result.Skipped = append(result.Skipped, path)
			continue
		}
		oursItem, err := codec.Parse(ours.Content)
		if err != nil {
			result.Skipped = append(result.Skipped, path)
			continue
		}
		if len(refs) > 0 && !itemMatchesAny(oursItem, refs) {
			continue
		}
		theirsItem, err := codec.Parse(theirs.Content)
		if err != nil {
			result.Skipped = append(result.Skipped, path)
			continue
		}
		if err := guardWritable(oursItem, theirsItem); err != nil {
			return nil, err
		}
		res := merge.Merge(parseOrNil(base.Content), oursItem, theirsItem, remote, gitTextMerge)
		if interactive {
			s.pickFields(res.Item, oursItem, theirsItem, res.Arbitrated)
		}
		if err := s.writeResolvedFile(path, res.Item); err != nil {
			return nil, err
		}
		staged = append(staged, path)
		result.Resolved = append(result.Resolved, mergeResultOf(res))
	}
	if len(staged) > 0 {
		if err := repo.Stage(staged...); err != nil {
			return nil, errx.User("%v", err)
		}
	}
	return result, nil
}

func (s *Store) writeResolvedFile(path string, it *datamodel.Item) error {
	abs := filepath.Join(s.root, filepath.FromSlash(path))
	if err := os.WriteFile(abs, []byte(codec.Serialize(it)), 0o644); err != nil {
		return errx.User("writing %s: %v", path, err)
	}
	return nil
}

func parseOrNil(content string) *datamodel.Item {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	it, err := codec.Parse(content)
	if err != nil {
		return nil
	}
	return it
}

func itemMatchesAny(it *datamodel.Item, refs []string) bool {
	for _, ref := range refs {
		if ref == it.Number || ref == it.ID || slices.Contains(it.Aliases, ref) {
			return true
		}
	}
	return false
}

func fieldString(it *datamodel.Item, field string) string {
	if d, ok := datamodel.Field(field); ok {
		return d.Get(it)
	}
	return ""
}

func applySide(dst, src *datamodel.Item, field string) {
	if d, ok := datamodel.Field(field); ok {
		d.Copy(dst, src)
	}
}

func (s *Store) pickFields(target, ours, theirs *datamodel.Item, fields []string) {
	for _, f := range fields {
		prompt := f + ": [o]urs=" + fieldString(ours, f) + " [t]heirs=" + fieldString(theirs, f) + " (default: auto) "
		switch strings.ToLower(s.prompter.ReadLine(prompt, "")) {
		case "o":
			applySide(target, ours, f)
		case "t":
			applySide(target, theirs, f)
		}
	}
}
