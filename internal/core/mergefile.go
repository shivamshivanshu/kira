package core

import (
	"os"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/merge"
	"github.com/shivamshivanshu/kira/internal/storage"
)

const itemFileMode = 0o644

type MergeSides struct{ Ours, Theirs string }

func MergeFile(repo gitx.Repo, basePath, oursPath, theirsPath string, sides MergeSides) (*datamodel.MergeResult, error) {
	store, err := DiscoverGitInvoked(repo.Dir)
	if err != nil {
		return nil, err
	}
	cfg, err := store.mergeDriverConfig(sides)
	if err != nil {
		return nil, err
	}

	baseData, _ := os.ReadFile(basePath)
	oursData, err := os.ReadFile(oursPath)
	if err != nil {
		return nil, errx.Conflict("reading %s: %v", oursPath, err)
	}
	theirsData, err := os.ReadFile(theirsPath)
	if err != nil {
		return nil, errx.Conflict("reading %s: %v", theirsPath, err)
	}

	if cfg.Merge.Policy == datamodel.MergeManual {
		return manualMergeFile(oursPath, string(baseData), string(oursData), string(theirsData))
	}

	ours, err := parseContent(oursData, oursPath)
	if err != nil {
		return nil, err
	}
	theirs, err := parseContent(theirsData, theirsPath)
	if err != nil {
		return nil, err
	}
	if err := guardWritable(ours, theirs); err != nil {
		return nil, err
	}
	base, err := parseBase(string(baseData), basePath)
	if err != nil {
		return nil, err
	}
	res := merge.Merge(base, ours, theirs, remoteSide(repo), gitTextMerge)
	if err := validateMergedWorkflow(cfg, res.Item, ours, theirs); err != nil {
		return nil, err
	}
	clearStaleResolution(cfg, res.Item)
	if err := os.WriteFile(oursPath, []byte(codec.Serialize(res.Item)), itemFileMode); err != nil {
		return nil, errx.Env("writing merge result: %v", err)
	}
	out := mergeResultOf(res)
	return &out, nil
}

func (s *Store) mergeDriverConfig(sides MergeSides) (*datamodel.Config, error) {
	if sides == (MergeSides{}) {
		return s.Config()
	}
	repo := s.repo()
	versions, err := repo.CatFileBatch([]string{
		gitx.RevPath(mergeSideRef(sides.Ours), storage.ConfigRelPath),
		gitx.RevPath(mergeSideRef(sides.Theirs), storage.ConfigRelPath),
	})
	if err != nil || !versions[0].Found || !versions[1].Found || versions[0].Content != versions[1].Content {
		return nil, unresolvedMergeConfig("the merge sides do not share a config")
	}
	worktree, err := os.ReadFile(s.fs().ConfigPath())
	if err != nil || string(worktree) != versions[0].Content {
		return nil, unresolvedMergeConfig("the worktree config differs from the merge sides")
	}
	return config.ParseCurrent(worktree)
}

func mergeSideRef(label string) string {
	// Rebase and cherry-pick label commits "<abbrev> (<subject>)".
	ref, _, _ := strings.Cut(label, " (")
	return ref + "^{commit}"
}

func unresolvedMergeConfig(reason string) error {
	return errx.Conflict("cannot merge ticket workflows: %s", reason).
		WithHint("finish resolving %s, then run kira resolve", storage.ConfigRelPath)
}

func manualMergeFile(oursPath, base, ours, theirs string) (*datamodel.MergeResult, error) {
	merged, conflict, err := gitx.MergeText(base, ours, theirs)
	if err != nil {
		return nil, errx.Conflict("merge.policy=manual text merge: %v", err)
	}
	if err := os.WriteFile(oursPath, []byte(merged), itemFileMode); err != nil {
		return nil, errx.Env("writing merge result: %v", err)
	}
	if conflict {
		return nil, errx.Conflict("merge.policy=manual left conflict markers in %s (resolve by hand or run kira resolve)", oursPath)
	}
	return &datamodel.MergeResult{}, nil
}

func mergeResultOf(res merge.Result) datamodel.MergeResult {
	return datamodel.MergeResult{
		ID:         res.Item.ID,
		Number:     res.Item.Number,
		Arbitrated: res.Arbitrated,
	}
}

func gitTextMerge(base, ours, theirs string) (string, bool) {
	merged, conflict, err := gitx.MergeText(base, ours, theirs)
	if err != nil {
		return "", true
	}
	return merged, conflict
}

func remoteSide(repo gitx.Repo) merge.Side {
	if repo.RebaseInProgress() {
		return merge.Ours
	}
	return merge.Theirs
}

func parseContent(data []byte, path string) (*datamodel.Item, error) {
	it, err := codec.Parse(string(data))
	if err != nil {
		return nil, errx.Conflict("%s is not a mergeable kira item: %v", path, err)
	}
	return it, nil
}
