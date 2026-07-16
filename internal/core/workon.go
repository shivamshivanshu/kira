package core

import (
	"fmt"
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/workon"
)

type WorkonOpts struct {
	NoMove   bool
	Worktree bool
}

func (s *Store) Workon(cfg *datamodel.Config, ref string, opts WorkonOpts) (*datamodel.WorkonResult, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	it, _, _, err := s.resolveRef(cfg, ref)
	if err != nil {
		return nil, err
	}
	repo := s.repo()
	branches, err := repo.Branches()
	if err != nil {
		return nil, errx.User("%v", err)
	}

	boardKey := boardKeyOf(it.Number)
	sep := cfg.Workon.Casing.Separator()
	branch, found := workon.MatchBranch(branches, cfg.Workon.BranchPattern, boardKey, it.Number, sep)
	if !found {
		branch = workon.RenderBranch(cfg.Workon.BranchPattern, boardKey, it.Number, it.Title, sep)
	}
	result := &datamodel.WorkonResult{ID: it.ID, Number: it.Number, Branch: branch}

	target := s
	if opts.Worktree {
		path, created, err := s.ensureWorktree(repo, cfg, it.Number, branch, !found)
		if err != nil {
			return nil, err
		}
		result.Worktree, result.BranchCreated = path, created
		target = newStore(path)
		target.prompter = s.prompter
	} else {
		created, err := checkoutBranch(repo, branch, found)
		if err != nil {
			return nil, err
		}
		result.BranchCreated = created
	}

	if err := target.writeActive(workon.ActivePointer{Ticket: it.ID, Branch: branch}); err != nil {
		return nil, err
	}

	if !opts.NoMove {
		if to, ok := doingTarget(cfg, it); ok {
			res, err := target.Move(cfg, ref, to, MoveOpts{})
			var msgs []string
			if err != nil {
				msgs = []string{fmt.Sprintf("workon: skipped doing-transition: %v", err)}
			} else {
				result.Moved, result.From, result.To = true, res.From, res.To
				msgs = res.Warnings
			}
			result.Warnings = literalWarnings(msgs)
		}
	}
	return result, nil
}

func checkoutBranch(repo gitx.Repo, branch string, exists bool) (created bool, err error) {
	if exists {
		if cur, _ := repo.CurrentBranch(); cur != branch {
			if err := repo.Checkout(branch); err != nil {
				return false, errx.User("%v", err)
			}
		}
		return false, nil
	}
	if err := repo.CheckoutNew(branch); err != nil {
		return false, errx.User("%v", err)
	}
	return true, nil
}

func (s *Store) ensureWorktree(repo gitx.Repo, cfg *datamodel.Config, number, branch string, createBranch bool) (string, bool, error) {
	if existing, ok := repo.WorktreeForBranch(branch); ok {
		return existing, false, nil
	}
	pattern := cfg.Workon.WorktreeDir
	if pattern == "" {
		pattern = datamodel.DefaultWorktreeDir
	}
	rendered := workon.RenderWorktreeDir(pattern, filepath.Base(s.root), branch, cfg.Project.Key, number)
	path := rendered
	if !filepath.IsAbs(rendered) {
		path = filepath.Join(s.root, rendered)
	}
	if err := repo.WorktreeAdd(path, branch, createBranch); err != nil {
		return "", false, errx.User("%v", err)
	}
	return path, createBranch, nil
}

func doingTarget(cfg *datamodel.Config, it *datamodel.Item) (string, bool) {
	wf, ok := cfg.Workflows[it.Type]
	if !ok {
		return "", false
	}
	for _, st := range wf.States {
		if st.Key == it.State && st.Category == datamodel.CategoryDoing {
			return "", false
		}
	}
	return firstStateInCategory(wf, datamodel.CategoryDoing)
}
