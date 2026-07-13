package core

import (
	"fmt"
	"os/exec"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

const subjectPrefix = "kira: "

func (s *Store) repo() gitx.Repo {
	return gitx.Repo{Dir: s.root}
}

func (s *Store) CommitShowCmd(sha string) *exec.Cmd { return s.repo().ShowCmd(sha) }

func (s *Store) requireRepo() error {
	if !gitx.Installed() {
		return errx.Env("git binary not found on PATH").WithHint("install git and make sure it is on your PATH")
	}
	if err := s.repo().InsideWorkTree(); err != nil {
		return errx.Env("not a git repository: %s", s.root).WithHint("run `git init` here first")
	}
	return nil
}

type commitSpec struct {
	trailerKey    string
	subject       string
	trailerNumber string
}

func (s *Store) finalize(mode datamodel.CommitMode, spec commitSpec, paths ...string) (string, error) {
	if err := s.repo().Stage(paths...); err != nil {
		return "", errx.User("%s", err)
	}
	switch mode {
	case datamodel.CommitManual:
		return "", nil
	case datamodel.CommitPrompt:
		if !s.prompter.Interactive() {
			return "", nil
		}
		if !s.prompter.Confirm(fmt.Sprintf("commit %q? [y/N] ", spec.subject)) {
			return "", nil
		}
		fallthrough
	default:
		if err := s.repo().Commit(spec.subject, spec.trailerKey, spec.trailerNumber); err != nil {
			return "", errx.User("%s", err)
		}
		sha, _ := s.repo().Output("rev-parse", "HEAD")
		return sha, nil
	}
}
