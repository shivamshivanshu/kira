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
	branch, found := workon.MatchBranch(branches, cfg.Workon.BranchPattern, boardKey, it.Number)
	if !found {
		branch = workon.RenderBranch(cfg.Workon.BranchPattern, boardKey, it.Number, it.Title)
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
		if err := target.moveWorkonTicket(result); err != nil {
			result.Warnings = literalWarnings([]string{fmt.Sprintf("workon: skipped doing-transition: %v", err)})
		}
	}
	return result, nil
}

func (s *Store) moveWorkonTicket(result *datamodel.WorkonResult) error {
	cfg, err := s.Config()
	if err != nil {
		return err
	}
	item, _, _, err := s.resolveRef(cfg, result.ID)
	if err != nil {
		return err
	}
	result.Number = item.Number
	target, ok := doingTarget(cfg, item)
	if !ok {
		return nil
	}
	move, err := s.Move(cfg, item.ID, target, MoveOpts{})
	if err != nil {
		return err
	}
	result.Moved, result.From, result.To = true, move.From, move.To
	result.Warnings = literalWarnings(move.Warnings)
	return nil
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
	if cat, ok := cfg.ItemCategory(it); ok && cat == datamodel.CategoryDoing {
		return "", false
	}
	return workflowFor(cfg, it).FirstInCategory(datamodel.CategoryDoing)
}
