package core

import (
	"fmt"
	"path/filepath"
	"strings"

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

	branch, found := workon.MatchBranch(branches, cfg.Workon.BranchPattern, cfg.Project.Key, it.Number, cfg.Workon.Casing)
	if !found {
		branch = workon.RenderBranch(cfg.Workon.BranchPattern, cfg.Project.Key, it.Number, it.Title, cfg.Workon.Casing)
	}
	result := &datamodel.WorkonResult{ID: it.ID, Number: it.Number, Branch: branch}

	target := s
	if opts.Worktree {
		path, created, err := s.ensureWorktree(repo, branch, !found)
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
			if err != nil {
				emitWarnings([]error{fmt.Errorf("workon: skipped doing-transition: %v", err)})
			} else {
				result.Moved, result.From, result.To = true, res.From, res.To
			}
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

func (s *Store) ensureWorktree(repo gitx.Repo, branch string, createBranch bool) (string, bool, error) {
	if existing, ok := repo.WorktreeForBranch(branch); ok {
		return existing, false, nil
	}
	path := filepath.Join(filepath.Dir(s.root), filepath.Base(s.root)+"-"+strings.ReplaceAll(branch, "/", "-"))
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
	for _, st := range wf.States {
		if st.Category == datamodel.CategoryDoing {
			return st.Key, true
		}
	}
	return "", false
}
