package core

import (
	"fmt"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/syncx"
)

type SyncOpts struct {
	Push   bool
	Dirty  syncx.DirtyPolicy
	Remote string
}

func (s *Store) Sync(cfg *datamodel.Config, opts SyncOpts, reindexer syncx.Reindexer) (report *syncx.Report, err error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	if reindexer == nil {
		reindexer = storeReindexer{store: s, cfg: cfg}
	}
	repo := s.repo()
	report = &syncx.Report{}

	stashed, perr := s.prepareTree(cfg, repo, opts, report)
	if perr != nil {
		return report, perr
	}
	if stashed {
		// Restore the stashed working tree on every exit path, so a sync that
		// fails after stashing never strands the user's edits on the stack.
		defer func() {
			if popErr := s.popStash(cfg, repo, report); popErr != nil && err == nil {
				err = popErr
			}
		}()
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

	reindexStep := reindexer.Reindex()
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
	s.fireSyncCompleted(cfg, report)
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

func (s *Store) prepareTree(cfg *datamodel.Config, repo gitx.Repo, opts SyncOpts, report *syncx.Report) (bool, error) {
	dirty, err := repo.DirtyPaths(".kira")
	if err != nil {
		return false, errx.User("%v", err)
	}
	if len(dirty) == 0 {
		report.Add("prepare", syncx.StepDone, "working tree clean")
		return false, nil
	}
	policy := opts.Dirty
	if policy == syncx.DirtyAuto {
		if cfg.Commit.Mode != datamodel.CommitAuto {
			return false, errx.User("uncommitted kira changes; re-run with --commit or --stash")
		}
		policy = syncx.DirtyCommit
	}
	if policy == syncx.DirtyStash {
		if err := repo.Stash(); err != nil {
			return false, errx.User("%v", err)
		}
		report.Add("prepare", syncx.StepDone, fmt.Sprintf("stashed %d paths", len(dirty)))
		return true, nil
	}
	if _, err := s.finalize(datamodel.CommitAuto, cfg.Commit.Trailer, "kira: sync checkpoint", "", dirty...); err != nil {
		return false, err
	}
	report.Add("prepare", syncx.StepDone, fmt.Sprintf("committed %d paths", len(dirty)))
	return false, nil
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
		repo.RebaseAbort()
		report.Add("pull", syncx.StepFailed, "merge.policy manual: conflicts left for you")
		return errx.Conflict("rebase halted with conflicts (merge.policy: manual)")
	}
	for range 100 {
		if _, err := s.Resolve(nil, false); err != nil {
			repo.RebaseAbort()
			return err
		}
		// A successful auto-resolve clears every modify/modify kira conflict, so
		// anything still unmerged (a non-kira path, or a kira file the engine
		// could not parse/apply) is terminal — abort at once naming it rather
		// than spinning until the iteration cap on unchanging state.
		if unmerged, _ := repo.UnmergedPaths(); len(unmerged) > 0 {
			repo.RebaseAbort()
			report.Add("pull", syncx.StepFailed, "unresolved conflicts: "+strings.Join(unmerged, ", "))
			return errx.Conflict("rebase halted, could not auto-resolve: %s", strings.Join(unmerged, ", "))
		}
		if err := repo.RebaseContinue(); err != nil && !repo.RebaseInProgress() {
			repo.RebaseAbort()
			report.Add("pull", syncx.StepFailed, err.Error())
			return errx.Conflict("%v", err)
		}
		if !repo.RebaseInProgress() {
			report.Add("pull", syncx.StepDone, "rebased with auto-resolved kira conflicts")
			return nil
		}
	}
	repo.RebaseAbort()
	report.Add("pull", syncx.StepFailed, "did not converge")
	return errx.Conflict("rebase did not converge")
}

func (s *Store) popStash(cfg *datamodel.Config, repo gitx.Repo, report *syncx.Report) error {
	if err := repo.StashPop(); err == nil {
		report.Add("stash-pop", syncx.StepDone, "")
		return nil
	}
	if cfg.Merge.Policy != datamodel.MergeAuto {
		report.Add("stash-pop", syncx.StepFailed, "conflicts on pop; resolve then `git stash drop`")
		return errx.Conflict("stash pop conflicted (merge.policy: manual); resolve, then `git stash drop`")
	}
	if _, err := s.Resolve(nil, false); err != nil {
		report.Add("stash-pop", syncx.StepFailed, err.Error())
		return errx.Conflict("stash pop conflicted and auto-resolve failed: %v", err)
	}
	if unmerged, _ := repo.UnmergedPaths(); len(unmerged) > 0 {
		report.Add("stash-pop", syncx.StepFailed, "unresolved: "+strings.Join(unmerged, ", "))
		return errx.Conflict("stash pop left unresolved conflicts: %s (resolve, then `git stash drop`)", strings.Join(unmerged, ", "))
	}
	report.Add("stash-pop", syncx.StepDone, "auto-resolved conflicts")
	return nil
}
