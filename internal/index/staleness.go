package index

import (
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

const (
	actionFresh       = "fresh"
	actionIncremental = "incremental"
	actionFull        = "full"
)

func (i *Index) EnsureFresh(store *storage.Store, repo gitx.Repo, opts Options) (Result, error) {
	return i.reindex(store, repo, opts, false)
}

func (i *Index) Rebuild(store *storage.Store, repo gitx.Repo, opts Options) (Result, error) {
	return i.reindex(store, repo, opts, true)
}

func (i *Index) reindex(store *storage.Store, repo gitx.Repo, opts Options, force bool) (Result, error) {
	toplevel, head, err := repo.ToplevelHead()
	if err != nil {
		return Result{}, err
	}
	root := gitx.Repo{Dir: toplevel}
	pathspec, err := filepath.Rel(toplevel, store.ItemsDir())
	if err != nil {
		return Result{}, errx.User("locating tickets under repo: %v", err)
	}

	statusPaths, err := root.StatusPorcelain(pathspec)
	if err != nil {
		return Result{}, err
	}
	dirtyHash, dirtyPaths := dirtyState(ticketAbsPaths(toplevel, statusPaths))

	prev, hasMeta := i.loadMeta()
	d, err := decide(root, toplevel, pathspec, force, hasMeta, head, prev, dirtyHash, dirtyPaths)
	if err != nil {
		return Result{}, err
	}
	warnings, err := i.dispatch(store, d)
	if err != nil {
		return Result{}, err
	}

	numbers, err := i.numberToULID()
	if err != nil {
		return Result{}, err
	}
	headCommits, watermarks, err := i.scanTrailers(root, opts, head, prev, numbers)
	if err != nil {
		return Result{}, err
	}
	var closes CloseScan
	if opts.Closes {
		closes, err = i.collectCloses(root, opts, prev, numbers, head, opts.LandedRef, headCommits)
		if err != nil {
			return Result{}, err
		}
	}

	count, err := i.count()
	if err != nil {
		return Result{}, err
	}
	if d.name != actionFresh {
		if err := i.saveMeta(meta{
			LastIndexedHeadSHA: head,
			DirtyHash:          dirtyHash,
			DirtyPaths:         dirtyPaths,
			TrailerWatermarks:  watermarks,
		}); err != nil {
			return Result{}, err
		}
	}
	return Result{Action: d.name, Reason: d.reason, Items: count, Closes: closes, Warnings: warnings}, nil
}

type decision struct {
	name    string
	reason  string
	full    bool
	refresh []string
}

func decide(root gitx.Repo, toplevel, pathspec string, force, hasMeta bool, head string, prev meta, dirtyHash string, dirtyPaths []string) (decision, error) {
	switch {
	case force:
		return decision{name: actionFull, reason: "forced", full: true}, nil
	case !hasMeta:
		return decision{name: actionFull, reason: "missing", full: true}, nil
	case head == prev.LastIndexedHeadSHA && dirtyHash == prev.DirtyHash:
		return decision{name: actionFresh, reason: "up-to-date"}, nil
	case head != prev.LastIndexedHeadSHA:
		anc := head != "" && prev.LastIndexedHeadSHA != ""
		if anc {
			ok, err := root.IsAncestor(prev.LastIndexedHeadSHA, head)
			if err != nil {
				return decision{}, err
			}
			anc = ok
		}
		if !anc {
			return decision{name: actionFull, reason: "history-rewritten", full: true}, nil
		}
		committed, err := root.DiffNameStatus(prev.LastIndexedHeadSHA, head, pathspec)
		if err != nil {
			return decision{}, err
		}
		refresh := unionPaths(ticketAbsPaths(toplevel, committed), unionPaths(dirtyPaths, prev.DirtyPaths))
		return decision{name: actionIncremental, reason: "head-advanced", refresh: refresh}, nil
	default:
		return decision{name: actionIncremental, reason: "working-tree-changed", refresh: unionPaths(dirtyPaths, prev.DirtyPaths)}, nil
	}
}

func (i *Index) dispatch(store *storage.Store, d decision) ([]string, error) {
	switch {
	case d.full:
		return i.full(store)
	case d.name == actionIncremental:
		return i.refresh(d.refresh)
	default:
		return nil, nil
	}
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

type FreshnessReport struct {
	Built  bool
	Fresh  bool
	Reason string
}

func Probe(store *storage.Store, repo gitx.Repo) (FreshnessReport, error) {
	prev, hasMeta := loadMetaAt(store.CacheDir())
	if !hasMeta {
		return FreshnessReport{}, nil
	}
	toplevel, head, err := repo.ToplevelHead()
	if err != nil {
		return FreshnessReport{}, err
	}
	root := gitx.Repo{Dir: toplevel}
	pathspec, err := filepath.Rel(toplevel, store.ItemsDir())
	if err != nil {
		return FreshnessReport{}, errx.User("locating tickets under repo: %v", err)
	}
	statusPaths, err := root.StatusPorcelain(pathspec)
	if err != nil {
		return FreshnessReport{}, err
	}
	dirtyHash, dirtyPaths := dirtyState(ticketAbsPaths(toplevel, statusPaths))
	d, err := decide(root, toplevel, pathspec, false, hasMeta, head, prev, dirtyHash, dirtyPaths)
	if err != nil {
		return FreshnessReport{}, err
	}
	return FreshnessReport{Built: true, Fresh: d.name == actionFresh, Reason: d.reason}, nil
}

func (i *Index) count() (int, error) {
	var n int
	if err := i.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&n); err != nil {
		return 0, errx.User("counting index items: %v", err)
	}
	return n, nil
}
