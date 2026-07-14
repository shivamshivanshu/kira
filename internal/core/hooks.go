package core

import (
	"os"
	"path/filepath"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/hooks"
)

const (
	filePerm = 0o644
	dirPerm  = 0o755
	execPerm = 0o755
)

type HooksInstallOpts struct {
	WithPreCommit bool
}

func (s *Store) InstallHooks(cfg *datamodel.Config, opts HooksInstallOpts) (*datamodel.HooksInstallResult, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	repo := s.repo()
	names := append([]string(nil), hooks.Default...)
	if opts.WithPreCommit {
		names = append(names, hooks.PreCommit)
	}

	result := &datamodel.HooksInstallResult{}
	var tracked []string
	for _, name := range names {
		script, ok := hooks.Script(name)
		if !ok {
			return nil, errx.User("no embedded script for hook %q", name)
		}
		trackedPath, err := s.materializeTrackedHook(name, script)
		if err != nil {
			return nil, err
		}
		tracked = append(tracked, trackedPath)

		status, err := s.installGitHook(repo, name, script)
		if err != nil {
			return nil, err
		}
		result.Hooks = append(result.Hooks, status)
	}

	if err := s.RegisterMergeDriver(); err != nil {
		return nil, err
	}
	result.MergeDriver = true

	if err := s.commitTrackedHooks(cfg, repo, tracked); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) materializeTrackedHook(name, script string) (string, error) {
	dir := filepath.Join(s.fs().KiraDir(), "hooks")
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return "", errx.User("creating %s: %v", s.fs().RelToRoot(dir), err)
	}
	path := filepath.Join(dir, name)
	if existing, err := os.ReadFile(path); err == nil && string(existing) == script {
		return path, nil
	}
	if err := os.WriteFile(path, []byte(script), execPerm); err != nil {
		return "", errx.User("writing %s: %v", s.fs().RelToRoot(path), err)
	}
	return path, nil
}

func (s *Store) installGitHook(repo gitx.Repo, name, script string) (datamodel.HookStatus, error) {
	status := datamodel.HookStatus{Name: name}
	dst, err := s.gitHookPath(repo, name)
	if err != nil {
		return status, err
	}
	if err := os.MkdirAll(filepath.Dir(dst), dirPerm); err != nil {
		return status, errx.User("creating %s: %v", filepath.Dir(dst), err)
	}

	existing, err := os.ReadFile(dst)
	if os.IsNotExist(err) {
		if err := writeExecutable(dst, script); err != nil {
			return status, err
		}
		status.Installed = true
		return status, nil
	}
	if err != nil {
		return status, errx.User("reading %s: %v", dst, err)
	}

	content := string(existing)
	if installed, chained := hooks.Classify(content, name); installed {
		status.Installed, status.Chained = true, chained
		return status, nil
	}
	if hooks.IsShellScript(content) {
		if err := writeExecutable(dst, hooks.Chain(content, name)); err != nil {
			return status, err
		}
		status.Installed, status.Chained = true, true
	}
	return status, nil
}

func (s *Store) gitHookPath(repo gitx.Repo, name string) (string, error) {
	p, err := repo.GitPath("hooks/" + name)
	if err != nil {
		return "", errx.User("%v", err)
	}
	return p, nil
}

func writeExecutable(path, content string) error {
	if err := os.WriteFile(path, []byte(content), execPerm); err != nil {
		return errx.User("writing %s: %v", path, err)
	}
	return os.Chmod(path, execPerm)
}

func (s *Store) commitTrackedHooks(cfg *datamodel.Config, repo gitx.Repo, tracked []string) error {
	dirty, err := repo.DirtyPaths(tracked...)
	if err != nil {
		return errx.User("%v", err)
	}
	if len(dirty) == 0 {
		return nil
	}
	_, err = s.finalize(cfg.Commit.Mode, commitSpec{subject: cfg.Commit.SubjectPrefix + "install hooks"}, tracked...)
	return err
}

func (s *Store) ValidateHooks(cfg *datamodel.Config) (*datamodel.HooksValidateResult, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	repo := s.repo()
	result := &datamodel.HooksValidateResult{OK: true}
	for _, name := range hooks.Default {
		status := datamodel.HookStatus{Name: name}
		if dst, err := s.gitHookPath(repo, name); err == nil {
			if content, err := os.ReadFile(dst); err == nil {
				status.Installed, status.Chained = hooks.Classify(string(content), name)
			}
		}
		if !status.Installed {
			result.OK = false
		}
		result.Hooks = append(result.Hooks, status)
	}
	result.MergeDriver = s.mergeDriverRegistered(repo)
	if !result.MergeDriver {
		result.OK = false
	}
	return result, nil
}

func (s *Store) mergeDriverRegistered(repo gitx.Repo) bool {
	if drv, err := repo.Output("config", "--get", "merge.kira.driver"); err != nil || drv == "" {
		return false
	}
	return repo.InfoAttributeHasLine(mergeAttrLine)
}
