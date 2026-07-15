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
	IntoHooksPath bool
}

func (s *Store) InstallHooks(cfg *datamodel.Config, opts HooksInstallOpts) (*datamodel.HooksInstallResult, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	repo := s.repo()
	if dir, override := s.hooksPathOverride(repo); override && !opts.IntoHooksPath {
		return nil, errx.User("this repo routes hooks through core.hooksPath (%s)", dir).
			WithHint("re-run with --into-hooks-path to install kira shims there")
	}
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
		return "", errx.Env("creating %s: %v", s.fs().RelToRoot(dir), err)
	}
	path := filepath.Join(dir, name)
	if existing, err := os.ReadFile(path); err == nil && string(existing) == script {
		return path, nil
	}
	if err := os.WriteFile(path, []byte(script), execPerm); err != nil {
		return "", errx.Env("writing %s: %v", s.fs().RelToRoot(path), err)
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
		return status, errx.Env("creating %s: %v", filepath.Dir(dst), err)
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
		return status, errx.Env("reading %s: %v", dst, err)
	}

	content := string(existing)
	switch hooks.StateOf(content, name) {
	case hooks.StateInstalled:
		status.Installed = true
	case hooks.StateChained:
		status.Installed, status.Chained = true, true
	case hooks.StateDrifted:
		if hooks.IsPureShim(content, name) {
			if err := writeExecutable(dst, script); err != nil {
				return status, err
			}
			status.Installed = true
			return status, nil
		}
		if _, chained := hooks.Classify(content, name); chained {
			status.Installed, status.Chained = true, true
			return status, nil
		}
		status.Note = "it already runs kira alongside other commands; remove kira's lines and re-run `kira hooks install`, or keep managing it by hand"
	default:
		if hooks.IsShellScript(content) {
			if err := writeExecutable(dst, hooks.Chain(content, name)); err != nil {
				return status, err
			}
			status.Installed, status.Chained = true, true
		}
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
		return errx.Env("writing %s: %v", path, err)
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
	if repo.ConfigValue("merge.kira.driver") == "" {
		return false
	}
	return repo.InfoAttributeHasLine(mergeAttrLine)
}

func (s *Store) hooksPathOverride(repo gitx.Repo) (string, bool) {
	val := repo.ConfigValue("core.hooksPath")
	if val == "" {
		return "", false
	}
	abs := val
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(s.root, val)
	}
	common, err := repo.Output("rev-parse", "--git-common-dir")
	if err != nil {
		return val, true
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(s.root, common)
	}
	if filepath.Clean(abs) == filepath.Join(filepath.Clean(common), "hooks") {
		return "", false
	}
	return val, true
}

func (s *Store) HooksStatus() (*datamodel.HooksStatusResult, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	repo := s.repo()
	result := &datamodel.HooksStatusResult{OK: true, HooksPath: repo.ConfigValue("core.hooksPath")}
	for _, name := range hooks.Known() {
		state := s.hookState(repo, name)
		if !stateOK(name, state) {
			result.OK = false
		}
		result.Hooks = append(result.Hooks, datamodel.HookState{Name: name, State: string(state)})
	}
	result.MergeDriver = s.mergeDriverRegistered(repo)
	if !result.MergeDriver {
		result.OK = false
	}
	return result, nil
}

func stateOK(name string, state hooks.State) bool {
	switch state {
	case hooks.StateInstalled, hooks.StateChained:
		return true
	case hooks.StateMissing:
		return name == hooks.PreCommit
	}
	return false
}

func (s *Store) hookState(repo gitx.Repo, name string) hooks.State {
	dst, err := s.gitHookPath(repo, name)
	if err != nil {
		return hooks.StateMissing
	}
	content, err := os.ReadFile(dst)
	if err != nil {
		return hooks.StateMissing
	}
	return hooks.StateOf(string(content), name)
}

func (s *Store) UninstallHooks() (*datamodel.HooksUninstallResult, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	repo := s.repo()
	result := &datamodel.HooksUninstallResult{}
	for _, name := range hooks.Known() {
		hs, err := s.uninstallGitHook(repo, name)
		if err != nil {
			return nil, err
		}
		result.Hooks = append(result.Hooks, hs)
	}
	unregistered, err := s.unregisterMergeDriver(repo)
	if err != nil {
		return nil, err
	}
	result.MergeDriver = unregistered
	return result, nil
}

func (s *Store) uninstallGitHook(repo gitx.Repo, name string) (datamodel.HookState, error) {
	hs := datamodel.HookState{Name: name}
	dst, err := s.gitHookPath(repo, name)
	if err != nil {
		return hs, err
	}
	existing, err := os.ReadFile(dst)
	if os.IsNotExist(err) {
		hs.State = "absent"
		return hs, nil
	}
	if err != nil {
		return hs, errx.Env("reading %s: %v", dst, err)
	}
	content := string(existing)
	state := hooks.StateOf(content, name)
	if state == hooks.StateInstalled || (state == hooks.StateDrifted && hooks.IsPureShim(content, name)) {
		if err := os.Remove(dst); err != nil {
			return hs, errx.Env("removing %s: %v", dst, err)
		}
		hs.State = "removed"
		return hs, nil
	}
	if stripped, changed := hooks.Unchain(content, name); changed {
		if err := writeExecutable(dst, stripped); err != nil {
			return hs, err
		}
		hs.State = "unchained"
		return hs, nil
	}
	hs.State = "left"
	if state == hooks.StateDrifted {
		hs.Note = dst + " still delegates to kira but was edited; remove kira's lines by hand"
	}
	return hs, nil
}

func (s *Store) unregisterMergeDriver(repo gitx.Repo) (bool, error) {
	had := repo.ConfigValue("merge.kira.driver") != "" || repo.InfoAttributeHasLine(mergeAttrLine)
	for _, key := range []string{"merge.kira.driver", "merge.kira.name"} {
		if err := repo.UnsetConfig(key); err != nil {
			return false, errx.User("%v", err)
		}
	}
	if err := repo.RemoveInfoAttribute(mergeAttrLine); err != nil {
		return false, errx.User("%v", err)
	}
	return had, nil
}
