package core

import (
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/merge"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func (s *Store) Resolve(cfg *datamodel.Config, refs []string, interactive bool) (*datamodel.ResolveResult, error) {
	if interactive && !s.prompter.Interactive() {
		return nil, errx.User("--interactive needs a terminal; rerun without it to auto-resolve").WithHint("run in an interactive shell to pick fields by hand")
	}
	release, err := s.fs().Lock()
	if err != nil {
		return nil, err
	}
	defer release()
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
		specs = append(specs, gitx.RevPath(":1", p), gitx.RevPath(":2", p), gitx.RevPath(":3", p))
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
		if s.fs().RelToRoot(s.fs().ItemPath(oursItem.ID)) != path {
			result.Skipped = append(result.Skipped, path)
			continue
		}
		res := merge.Merge(parseOrNil(base.Content), oursItem, theirsItem, remote, gitTextMerge)
		if interactive {
			s.pickFields(res.Item, oursItem, theirsItem, res.Arbitrated)
		}
		clearStaleResolution(cfg, res.Item)
		rel, err := s.fs().WriteItemRaw(res.Item.ID, codec.Serialize(res.Item))
		if err != nil {
			return nil, err
		}
		staged = append(staged, rel)
		result.Resolved = append(result.Resolved, mergeResultOf(res))
	}
	if len(staged) > 0 {
		if err := repo.Stage(staged...); err != nil {
			return nil, errx.User("%v", err)
		}
	}
	return result, nil
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

// RefMatches reports whether ref identifies it, by display number, ULID, or alias.
func RefMatches(it *datamodel.Item, ref string) bool {
	return ref == it.Number || ref == it.ID || slices.Contains(it.Aliases, ref)
}

func itemMatchesAny(it *datamodel.Item, refs []string) bool {
	for _, ref := range refs {
		if RefMatches(it, ref) {
			return true
		}
	}
	return false
}

func fieldString(it *datamodel.Item, field string) string {
	if d, ok := datamodel.Field(field); ok {
		return d.Get(it)
	}
	if v, ok := it.ExtraScalars[field]; ok {
		return codec.DecodeScalar(v)
	}
	if v, ok := it.ExtraLists[field]; ok {
		return strings.Join(codec.DecodeExtraList(v), ", ")
	}
	return "-"
}

func (s *Store) pickFields(target, ours, theirs *datamodel.Item, fields []string) {
	for _, f := range fields {
		prompt := f + ": [o]urs=" + fieldString(ours, f) + " [t]heirs=" + fieldString(theirs, f) + " (default: auto) "
		switch strings.ToLower(s.prompter.ReadLine(prompt, "")) {
		case "o":
			applyPick(target, ours, f)
		case "t":
			applyPick(target, theirs, f)
		}
	}
}

func applyPick(target, src *datamodel.Item, field string) {
	if d, ok := datamodel.Field(field); ok {
		d.Copy(target, src)
		return
	}
	copyExtra(target, src, field)
}

func copyExtra(target, src *datamodel.Item, key string) {
	delete(target.ExtraScalars, key)
	delete(target.ExtraLists, key)
	if v, ok := src.ExtraScalars[key]; ok {
		if target.ExtraScalars == nil {
			target.ExtraScalars = map[string]datamodel.Rendered{}
		}
		target.ExtraScalars[key] = v
		return
	}
	if v, ok := src.ExtraLists[key]; ok {
		if target.ExtraLists == nil {
			target.ExtraLists = map[string][]datamodel.Rendered{}
		}
		target.ExtraLists[key] = slices.Clone(v)
	}
}
