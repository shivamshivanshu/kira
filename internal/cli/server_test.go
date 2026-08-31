package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

func TestServerAddress(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:7432", "localhost:0", "[::1]:7432", "127.0.0.2:1234"} {
		if _, err := serverAddress(addr); err != nil {
			t.Errorf("serverAddress(%q): %v", addr, err)
		}
	}
	for _, addr := range []string{"0.0.0.0:7432", ":7432", "[::]:7432", "192.168.1.2:7432", "example.com:7432", "127.0.0.1:-1", "127.0.0.1:65536", "localhost:http", "localhost"} {
		if _, err := serverAddress(addr); err == nil {
			t.Errorf("serverAddress(%q) must fail", addr)
		}
	}
}

func TestServerCommandRunnerUsesStore(t *testing.T) {
	dir := initFixture(t)
	run := serverCommandRunner(dir)
	if out, err := run([]string{"create", "ticket", "--title", "Dashboard ticket"}); err != nil {
		t.Fatalf("create: %v\n%s", err, out)
	}
	s, cfg := reopen(t, dir)
	items, err := s.List(cfg, core.ListOpts{})
	if err != nil || len(items.Items) != 1 {
		t.Fatalf("create in pinned store: %+v, %v", items, err)
	}
	ref := items.Items[0].Number
	if _, err := run([]string{"move", ref, "IN_PROGRESS"}); err != nil {
		t.Fatal(err)
	}
	if state := stateOf(t, s, cfg, ref); state != "IN_PROGRESS" {
		t.Fatalf("state = %q", state)
	}
	for _, args := range [][]string{{"show", ref}, {"list"}, {"board"}, {"config", "filters"}, {"find", "Dashboard", "-i"}, {"hooks", "status"}, {"automation", "list"}} {
		if _, err := run(args); err != nil {
			t.Errorf("%v: %v", args, err)
		}
	}
}

func TestServerCommandRunnerConfigSet(t *testing.T) {
	dir := initFixture(t)
	run := serverCommandRunner(dir)
	if out, err := run([]string{"config", "set", "project.name", "Browser workspace"}); err != nil {
		t.Fatalf("config set: %v\n%s", err, out)
	}
	_, cfg := reopen(t, dir)
	if cfg.Project.Name != "Browser workspace" {
		t.Fatalf("project name = %q", cfg.Project.Name)
	}
	if _, err := run([]string{"config", "set", "commit.mode", "unsupported"}); err == nil {
		t.Fatal("config set must retain core validation")
	}
	_, cfg = reopen(t, dir)
	if cfg.Commit.Mode != datamodel.CommitAuto {
		t.Fatalf("invalid config edit changed commit mode to %q", cfg.Commit.Mode)
	}
}

func TestServerCommandRunnerWorkonStaysInCheckout(t *testing.T) {
	dir := initFixture(t)
	xdg, worktrees := t.TempDir(), t.TempDir()
	userDir := filepath.Join(xdg, "kira")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	userConfig := fmt.Sprintf("workon:\n  worktree: true\n  worktree_dir: %q\n", filepath.Join(worktrees, "{number}"))
	if err := os.WriteFile(filepath.Join(userDir, "config.yaml"), []byte(userConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)
	s, cfg := reopen(t, dir)
	if !cfg.Workon.Worktree {
		t.Fatal("fixture must inherit worktree mode from user configuration")
	}
	run := serverCommandRunner(dir)
	for _, flag := range []string{"", "--worktree=false"} {
		it, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "Local branch " + flag, NoEdit: true})
		if err != nil {
			t.Fatal(err)
		}
		args := []string{"workon", it.Number, "--json"}
		if flag != "" {
			args = append(args, flag)
		}
		output, err := run(args)
		if err != nil {
			t.Fatalf("workon %q: %v\n%s", flag, err, output)
		}
		var result datamodel.WorkonResult
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			t.Fatalf("workon JSON: %v\n%s", err, output)
		}
		branch, err := (gitx.Repo{Dir: dir}).CurrentBranch()
		if err != nil {
			t.Fatal(err)
		}
		if result.Worktree != "" || result.Branch != branch || !result.Moved || stateOf(t, s, cfg, it.Number) != "IN_PROGRESS" {
			t.Fatalf("workon did not stay in checkout: %+v, current branch %q", result, branch)
		}
	}
	listed, err := (gitx.Repo{Dir: dir}).Output("worktree", "list", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(listed, "worktree ") != 1 {
		t.Fatalf("workon created another worktree:\n%s", listed)
	}
	entries, err := os.ReadDir(worktrees)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("workon wrote to external worktree directory: %v", entries)
	}
}

func TestServerCommandRunnerSync(t *testing.T) {
	dir := initFixture(t)
	run := serverCommandRunner(dir)
	output, err := run([]string{"sync"})
	if err == nil || !strings.Contains(output, "pull:") || !strings.Contains(output, "failed") {
		t.Fatalf("sync must surface the core failure without an upstream: %v\n%s", err, output)
	}
	remoteDir := t.TempDir()
	if _, err := (gitx.Repo{Dir: remoteDir}).Output("init", "--bare"); err != nil {
		t.Fatal(err)
	}
	repo := gitx.Repo{Dir: dir}
	for _, args := range [][]string{{"remote", "add", "origin", remoteDir}, {"push", "-u", "origin", "HEAD:main"}} {
		if _, err := repo.Output(args...); err != nil {
			t.Fatal(err)
		}
	}
	output, err = run([]string{"sync", "--remote", "origin"})
	if err != nil || !strings.Contains(output, "rebased") || !strings.Contains(output, "reindex:") {
		t.Fatalf("sync with a configured local remote: %v\n%s", err, output)
	}
}

func TestServerCommandRunnerRejectsEscapeRoutes(t *testing.T) {
	dir := initFixture(t)
	run := serverCommandRunner(dir)
	for _, args := range [][]string{
		nil, {"--C", dir, "list"}, {"list", "--C", dir}, {"list", "-C" + dir},
		{"list", "--non-interactive=false"}, {"tui"}, {"server"}, {"serve"},
		{"workon", "KIRA-1", "--worktree"}, {"workon", "KIRA-1", "--worktree=true"},
		{"workon", "KIRA-1", "-C", dir}, {"workon", "KIRA-1", "--non-interactive=false"},
		{"sync", "--remote", "--upload-pack=sh"}, {"sync", "--remote", "/tmp/repo"},
		{"sync", "--remote", "https://example.com/repo"}, {"sync", "--remote", "missing"},
		{"init"}, {"config", "init"}, {"config", "set", "ui.editor", "sh"},
		{"hooks", "install"}, {"hooks", "uninstall"}, {"hooks", "run", "post-merge"}, {"automation", "trust"},
		{"validate", "/tmp/item"},
		{"create", "ticket", "--from-file=/tmp/item"}, {"edit", "KIRA-1", "--from-file", "-"},
		{"find", "pattern", "/tmp"}, {"find", "pattern", "--pre=sh"},
		{"find", "pattern", "-f/tmp/secret"}, {"find", "pattern", "--hostname-bin=sh"},
		{"find", "pattern", "--non-interactive=false"}, {"find", "pattern", "-C3"},
	} {
		if _, err := run(args); err == nil {
			t.Errorf("%v must be rejected", args)
		}
	}
	if out, err := run([]string{"create", "ticket", "--title=-C"}); err != nil || !strings.Contains(out, "Created") {
		t.Fatalf("literal flag-like title must remain valid: %q, %v", out, err)
	}
}
