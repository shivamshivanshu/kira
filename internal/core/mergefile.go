package core

import (
	"os"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/merge"
)

func MergeFile(repo gitx.Repo, basePath, oursPath, theirsPath string) (*datamodel.MergeResult, error) {
	store, err := Discover(repo.Dir)
	if err != nil {
		return nil, err
	}
	cfg, err := store.Config()
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
	if err := guardKnownFields(ours, theirs); err != nil {
		return nil, err
	}
	res := merge.Merge(parseOrNil(string(baseData)), ours, theirs, remoteSide(repo), gitTextMerge)
	if err := os.WriteFile(oursPath, []byte(codec.Serialize(res.Item)), 0o644); err != nil {
		return nil, errx.User("writing merge result: %v", err)
	}
	out := mergeResultOf(res)
	return &out, nil
}

func manualMergeFile(oursPath, base, ours, theirs string) (*datamodel.MergeResult, error) {
	merged, conflict, err := gitx.MergeText(base, ours, theirs)
	if err != nil {
		return nil, errx.Conflict("merge.policy=manual text merge: %v", err)
	}
	if err := os.WriteFile(oursPath, []byte(merged), 0o644); err != nil {
		return nil, errx.User("writing merge result: %v", err)
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
