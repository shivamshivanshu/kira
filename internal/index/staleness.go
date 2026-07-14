package index

import (
	"maps"
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

type Result struct {
	Action   string
	Reason   string
	Items    int
	Closes   CloseScan
	Warnings []string
}

type action string

const (
	actionFresh       action = "fresh"
	actionIncremental action = "incremental"
	actionFull        action = "full"
)

func (i *Index) EnsureFresh(store *storage.FS, repo gitx.Repo, opts Options) (Result, error) {
	return i.reindex(store, repo, opts, false)
}

func (i *Index) reindex(store *storage.FS, repo gitx.Repo, opts Options, force bool) (Result, error) {
	prev, hasMeta := i.loadMeta()
	p, err := plan(store, repo, force, hasMeta, prev)
	if err != nil {
		return Result{}, err
	}
	warnings, err := i.dispatch(store, p.decision)
	if err != nil {
		return Result{}, err
	}

	numbers, err := i.numberToULID()
	if err != nil {
		return Result{}, err
	}
	configHash := scanConfigHash(opts)
	configChanged := hasMeta && prev.ScanConfigHash != configHash
	if p.decision.name == actionFull || configChanged {
		prev.TrailerWatermarks = withoutTrailerRef(prev.TrailerWatermarks)
	}
	headCommits, watermarks, err := i.scanTrailers(p.root, opts, p.head, prev, numbers)
	if err != nil {
		return Result{}, err
	}
	var closes CloseScan
	if opts.Closes {
		closes, err = i.collectCloses(p.root, opts, prev, numbers, p.head, headCommits)
		if err != nil {
			return Result{}, err
		}
	}

	if p.decision.name != actionFresh || configChanged {
		if err := i.fillActivity(); err != nil {
			return Result{}, err
		}
	}

	count, err := i.count()
	if err != nil {
		return Result{}, err
	}
	if p.decision.name != actionFresh || configChanged {
		if err := i.saveMeta(meta{
			ScanConfigHash:     configHash,
			LastIndexedHeadSHA: p.head,
			DirtyHash:          p.dirtyHash,
			DirtyPaths:         p.dirtyPaths,
			TrailerWatermarks:  watermarks,
		}); err != nil {
			return Result{}, err
		}
	}
	return Result{Action: string(p.decision.name), Reason: p.decision.reason, Items: count, Closes: closes, Warnings: warnings}, nil
}

type planResult struct {
	root       gitx.Repo
	head       string
	dirtyHash  string
	dirtyPaths []string
	decision   decision
}

func plan(store *storage.FS, repo gitx.Repo, force, hasMeta bool, prev meta) (planResult, error) {
	toplevel, head, err := repo.ToplevelHead()
	if err != nil {
		return planResult{}, err
	}
	root := gitx.Repo{Dir: toplevel}
	pathspec, err := filepath.Rel(toplevel, store.ItemsDir())
	if err != nil {
		return planResult{}, errx.User("locating tickets under repo: %v", err)
	}
	statusPaths, err := root.StatusPorcelain(pathspec)
	if err != nil {
		return planResult{}, err
	}
	dirtyHash, dirtyPaths := dirtyState(ticketAbsPaths(toplevel, statusPaths))
	d, err := decide(root, toplevel, pathspec, force, hasMeta, head, prev, dirtyHash, dirtyPaths)
	if err != nil {
		return planResult{}, err
	}
	return planResult{root: root, head: head, dirtyHash: dirtyHash, dirtyPaths: dirtyPaths, decision: d}, nil
}

type decision struct {
	name    action
	reason  string
	refresh []string
}

func decide(root gitx.Repo, toplevel, pathspec string, force, hasMeta bool, head string, prev meta, dirtyHash string, dirtyPaths []string) (decision, error) {
	switch {
	case force:
		return decision{name: actionFull, reason: "forced"}, nil
	case !hasMeta:
		return decision{name: actionFull, reason: "missing"}, nil
	case head == prev.LastIndexedHeadSHA && dirtyHash == prev.DirtyHash:
		return decision{name: actionFresh, reason: "up-to-date"}, nil
	case head != prev.LastIndexedHeadSHA:
		anc := head != "" && prev.LastIndexedHeadSHA != ""
		if anc {
			ok, err := root.IsAncestor(gitx.Ancestor(prev.LastIndexedHeadSHA), gitx.Descendant(head))
			if err != nil {
				return decision{}, err
			}
			anc = ok
		}
		if !anc {
			return decision{name: actionFull, reason: "history-rewritten"}, nil
		}
		committed, err := root.DiffNameStatus(gitx.DiffFrom(prev.LastIndexedHeadSHA), gitx.DiffTo(head), pathspec)
		if err != nil {
			return decision{}, err
		}
		refresh := unionPaths(ticketAbsPaths(toplevel, committed), unionPaths(dirtyPaths, prev.DirtyPaths))
		return decision{name: actionIncremental, reason: "head-advanced", refresh: refresh}, nil
	default:
		return decision{name: actionIncremental, reason: "working-tree-changed", refresh: unionPaths(dirtyPaths, prev.DirtyPaths)}, nil
	}
}

func (i *Index) dispatch(store *storage.FS, d decision) ([]string, error) {
	switch {
	case d.name == actionFull:
		return i.full(store)
	case d.name == actionIncremental:
		return i.refresh(d.refresh)
	default:
		return nil, nil
	}
}

func withoutTrailerRef(wm map[string]string) map[string]string {
	if wm == nil {
		return nil
	}
	out := maps.Clone(wm)
	delete(out, trailerRef)
	return out
}

func ticketAbsPaths(toplevel string, relPaths []string) []string {
	var out []string
	for _, rel := range relPaths {
		abs := filepath.Join(toplevel, rel)
		if storage.ULIDFromPath(abs) != "" {
			out = append(out, abs)
		}
	}
	return out
}

func unionPaths(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, group := range [][]string{a, b} {
		for _, p := range group {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

type FreshnessReport struct {
	Built  bool
	Fresh  bool
	Reason string
}

func Probe(store *storage.FS, repo gitx.Repo) (FreshnessReport, error) {
	prev, hasMeta := loadMetaAt(store.CacheDir())
	if !hasMeta {
		return FreshnessReport{}, nil
	}
	p, err := plan(store, repo, false, hasMeta, prev)
	if err != nil {
		return FreshnessReport{}, err
	}
	return FreshnessReport{Built: true, Fresh: p.decision.name == actionFresh, Reason: p.decision.reason}, nil
}

func (i *Index) count() (int, error) {
	var n int
	if err := i.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&n); err != nil {
		return 0, errx.User("counting index items: %v", err)
	}
	return n, nil
}
