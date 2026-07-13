package core

import (
	"fmt"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/termx"
)

func (s *Store) repo() gitx.Repo {
	return gitx.Repo{Dir: s.root}
}

func (s *Store) requireRepo() error {
	if !gitx.Installed() {
		return errx.Env("git binary not found on PATH").WithHint("install git and make sure it is on your PATH")
	}
	if err := s.repo().InsideWorkTree(); err != nil {
		return errx.Env("not a git repository: %s", s.root).WithHint("run `git init` here first")
	}
	return nil
}

func (s *Store) finalize(mode datamodel.CommitMode, trailerKey, subject, trailerNumber string, paths ...string) error {
	if err := s.repo().Stage(paths...); err != nil {
		return errx.User("%s", err)
	}
	switch mode {
	case datamodel.CommitManual:
		return nil
	case datamodel.CommitPrompt:
		if !termx.IsInteractive() {
			return nil
		}
		if !termx.Confirm(fmt.Sprintf("commit %q? [y/N] ", subject)) {
			return nil
		}
		fallthrough
	default:
		if err := s.repo().Commit(subject, trailerKey, trailerNumber); err != nil {
			return errx.User("%s", err)
		}
		return nil
	}
}
