package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/hooks"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func gatherEnv(root string) doctor.Env {
	env := doctor.Env{GitInstalled: gitx.Installed()}
	if !env.GitInstalled {
		return env
	}
	env.MissingOptionalBins = missingBinaries("rg", "fzf")

	repo := gitx.Repo{Dir: root}
	env.InsideWorkTree = repo.InsideWorkTree() == nil
	if !env.InsideWorkTree {
		return env
	}
	gitDir := gitDir(repo, root)
	env.TrackedHooks = trackedHooks(root)
	env.InstalledHooks = installedHooks(gitDir, env.TrackedHooks)
	_, err := repo.Output("config", "--get", "merge.kira.driver")
	env.MergeDriverRegistered = err == nil
	env.TicketAttrRegistered = fileContains(filepath.Join(gitDir, "info", "attributes"), "merge=kira")
	env.Freshness = doctor.ResolveFreshness(indexFreshnessReporter(root, repo))
	return env
}

func gitDir(repo gitx.Repo, root string) string {
	p, err := repo.Output("rev-parse", "--git-dir")
	if err != nil {
		return filepath.Join(root, ".git")
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	return p
}

type indexFreshness struct {
	store *storage.FS
	repo  gitx.Repo
}

func (r indexFreshness) Freshness() (doctor.Freshness, error) {
	rep, err := index.Probe(r.store, r.repo)
	if err != nil {
		return doctor.Freshness{}, err
	}
	return doctor.Freshness{Built: rep.Built, Fresh: rep.Fresh, Reason: rep.Reason}, nil
}

func indexFreshnessReporter(root string, repo gitx.Repo) doctor.FreshnessReporter {
	return indexFreshness{store: storage.New(root), repo: repo}
}

func missingBinaries(names ...string) []string {
	var out []string
	for _, n := range names {
		if _, err := exec.LookPath(n); err != nil {
			out = append(out, n)
		}
	}
	return out
}

func trackedHooks(root string) []string {
	dir := filepath.Join(storage.New(root).KiraDir(), "hooks")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

func installedHooks(gitDir string, tracked []string) []string {
	var out []string
	for _, name := range tracked {
		data, err := os.ReadFile(filepath.Join(gitDir, "hooks", name))
		if err != nil {
			continue
		}
		if installed, _ := hooks.Classify(string(data), name); installed {
			out = append(out, name)
		}
	}
	return out
}

func fileContains(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}
