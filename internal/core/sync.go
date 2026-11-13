package core

import (
	"fmt"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
	"github.com/shivamshivanshu/kira/internal/syncx"
)

const maxRebaseIterations = 100

type SyncOpts struct {
	Push   bool
	Dirty  syncx.DirtyPolicy
	Remote string
}

// Sync pulls and rebases against the remote, reconciles and reindexes local
// tickets, and optionally pushes, reporting the outcome of each step.
func (s *Store) Sync(cfg *datamodel.Config, opts SyncOpts) (*syncx.Report, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	repo := s.repo()
	report := &syncx.Report{}

	if e := s.prepareTree(repo, opts, report); e != nil {
		return report, e
	}

	if e := s.pullRebase(cfg, repo, opts, report); e != nil {
		return report, e
	}

	rec, e := s.Reconcile(cfg)
	if e != nil {
		report.Add("reconcile", syncx.StepFailed, e.Error())
		return report, e
	}
	detail := "no collisions"
	if n := len(rec.Renumbered); n > 0 {
		detail = fmt.Sprintf("%d renumbered", n)
	}
	report.Add("reconcile", syncx.StepDone, detail)

	reindexStep := storeReindexer{store: s, cfg: cfg}.Reindex()
	report.Steps = append(report.Steps, reindexStep)
	if reindexStep.Status == syncx.StepDone {
		report.Steps = append(report.Steps, s.syncCloses(cfg))
	}

	if opts.Push || cfg.Sync.Push {
		if e := repo.Push(opts.Remote); e != nil {
			report.Add("push", syncx.StepFailed, e.Error())
			return report, errx.User("%v", e)
		}
		report.Add("push", syncx.StepDone, "")
	}
	return report, nil
}

type storeReindexer struct {
	store *Store
	cfg   *datamodel.Config
}

func (r storeReindexer) Reindex() syncx.Step {
	res, err := index.Refresh(r.store.fs(), r.store.repo(), indexOptions(r.cfg), false)
	if err != nil {
		return syncx.Step{Name: "reindex", Status: syncx.StepFailed, Detail: err.Error()}
	}
	return syncx.Step{Name: "reindex", Status: syncx.StepDone, Detail: fmt.Sprintf("%d items", res.Items)}
}

func (s *Store) syncCloses(cfg *datamodel.Config) syncx.Step {
	res, err := s.Index(cfg, false, true)
	if err != nil {
		return syncx.Step{Name: "closes", Status: syncx.StepFailed, Detail: err.Error()}
	}
	if len(res.Closed) == 0 {
		return syncx.Step{Name: "closes", Status: syncx.StepDone, Detail: "no transitions"}
	}
	return syncx.Step{Name: "closes", Status: syncx.StepDone, Detail: fmt.Sprintf("closed %s", strings.Join(res.Closed, ", "))}
}

func (s *Store) prepareTree(repo gitx.Repo, opts SyncOpts, report *syncx.Report) error {
	dirty, err := repo.DirtyPaths(storage.DirName)
	if err != nil {
		return errx.User("%v", err)
	}
	if len(dirty) == 0 {
		report.Add("prepare", syncx.StepDone, "working tree clean")
		return nil
	}
	if opts.Dirty != syncx.DirtyCommit {
		return errx.User("uncommitted kira changes; re-run with --commit")
	}
	if _, err := s.finalize(datamodel.CommitAuto, commitSpec{subject: "sync checkpoint"}, dirty...); err != nil {
		return err
	}
	report.Add("prepare", syncx.StepDone, fmt.Sprintf("committed %d paths", len(dirty)))
	return nil
}

func (s *Store) pullRebase(cfg *datamodel.Config, repo gitx.Repo, opts SyncOpts, report *syncx.Report) error {
	err := repo.Pull(opts.Remote)
	if err == nil {
		report.Add("pull", syncx.StepDone, "rebased")
		return nil
	}
	if !repo.RebaseInProgress() {
		report.Add("pull", syncx.StepFailed, err.Error())
		return errx.Conflict("%v", err)
	}
	if cfg.Merge.Policy != datamodel.MergeAuto {
		_ = repo.RebaseAbort()
		report.Add("pull", syncx.StepFailed, "merge.policy manual: conflicts left for you")
		return errx.Conflict("rebase halted with conflicts (merge.policy: manual)")
	}
	for range maxRebaseIterations {
		unmerged, err := s.autoResolve(repo)
		if err != nil {
			_ = repo.RebaseAbort()
			return err
		}
		// A successful auto-resolve clears every modify/modify kira conflict, so
		// anything still unmerged (a non-kira path, or a kira file the engine
		// could not parse/apply) is terminal — abort at once naming it rather
		// than spinning until the iteration cap on unchanging state.
		if len(unmerged) > 0 {
			_ = repo.RebaseAbort()
			report.Add("pull", syncx.StepFailed, "unresolved conflicts: "+strings.Join(unmerged, ", "))
			return errx.Conflict("rebase halted, could not auto-resolve: %s", strings.Join(unmerged, ", "))
		}
		if err := repo.RebaseContinue(); err != nil {
			if !repo.RebaseInProgress() {
				_ = repo.RebaseAbort()
				report.Add("pull", syncx.StepFailed, err.Error())
				return errx.Conflict("%v", err)
			}
			// Auto-resolve can arbitrate every conflicting field to the incoming
			// side, leaving the staged file identical to the rebase parent; the
			// apply backend's --continue then refuses the empty patch. Skip it —
			// but only with an empty index, so a genuine failure while real
			// changes are staged never silently drops a commit.
			if staged, serr := repo.HasStagedChanges(); serr == nil && !staged {
				if skipErr := repo.RebaseSkip(); skipErr != nil {
					_ = repo.RebaseAbort()
					report.Add("pull", syncx.StepFailed, skipErr.Error())
					return errx.Conflict("%v", skipErr)
				}
			}
		}
		if !repo.RebaseInProgress() {
			report.Add("pull", syncx.StepDone, "rebased with auto-resolved kira conflicts")
			return nil
		}
	}
	_ = repo.RebaseAbort()
	report.Add("pull", syncx.StepFailed, "did not converge")
	return errx.Conflict("rebase did not converge")
}

func (s *Store) autoResolve(repo gitx.Repo) ([]string, error) {
	if _, err := s.Resolve(nil, false); err != nil {
		return nil, err
	}
	unmerged, _ := repo.UnmergedPaths()
	return unmerged, nil
}
