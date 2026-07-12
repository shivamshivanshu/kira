package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/item"
)

// initGitRepo creates an isolated git repo in a temp dir. Global/system git
// config is neutralized so a developer's real ~/.gitconfig (hooks, templates)
// cannot perturb the test.
func initGitRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("EDITOR", "true")
	root := t.TempDir()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "tester"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func commitCount(t *testing.T, root string) int {
	t.Helper()
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-list: %v", err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("parse count %q: %v", out, err)
	}
	return n
}

// TestAutoModeOneCommitPerMutation is the core M0 guarantee: init and each
// mutation produce exactly one commit.
func TestAutoModeOneCommitPerMutation(t *testing.T) {
	root := initGitRepo(t)
	if _, err := Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if got := commitCount(t, root); got != 1 {
		t.Fatalf("after init: %d commits, want 1", got)
	}
	s, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}

	res, err := s.Create(cfg, CreateOpts{Type: item.TypeTicket, Title: "First", NoEdit: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Number != "KIRA-1" {
		t.Fatalf("number = %q, want KIRA-1", res.Number)
	}
	if got := commitCount(t, root); got != 2 {
		t.Fatalf("after create: %d commits, want 2", got)
	}

	if _, err := s.Edit(cfg, "KIRA-1", EditOpts{Fields: []FieldEdit{{Key: "state", Value: "IN_PROGRESS"}}}); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if got := commitCount(t, root); got != 3 {
		t.Fatalf("after edit: %d commits, want 3", got)
	}

	// A no-op edit neither writes nor commits.
	if r, err := s.Edit(cfg, "KIRA-1", EditOpts{Fields: []FieldEdit{{Key: "state", Value: "IN_PROGRESS"}}}); err != nil {
		t.Fatalf("no-op Edit: %v", err)
	} else if len(r.Changed) != 0 {
		t.Fatalf("no-op edit reported changes: %v", r.Changed)
	}
	if got := commitCount(t, root); got != 3 {
		t.Fatalf("after no-op edit: %d commits, want 3", got)
	}
}

// TestManualModeStagesNoCommit checks manual mode stages the write but records
// no commit, leaving a dirty index.
func TestManualModeStagesNoCommit(t *testing.T) {
	root := initGitRepo(t)
	if _, err := Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	cfgPath := filepath.Join(root, ".kira", "config.yaml")
	data, _ := os.ReadFile(cfgPath)
	os.WriteFile(cfgPath, []byte(strings.Replace(string(data), "mode: auto", "mode: manual", 1)), 0o644)
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("add", ".kira/config.yaml")
	run("commit", "-m", "manual mode")
	before := commitCount(t, root)

	s, _ := Discover(root)
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if cfg.Commit.Mode != config.CommitManual {
		t.Fatalf("mode = %q, want manual", cfg.Commit.Mode)
	}
	res, err := s.Create(cfg, CreateOpts{Type: item.TypeTicket, Title: "Staged", NoEdit: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := commitCount(t, root); got != before {
		t.Fatalf("manual mode committed: count %d, want %d", got, before)
	}
	// The written file is staged.
	cmd := exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = root
	out, _ := cmd.Output()
	if !strings.Contains(string(out), res.Path) {
		t.Fatalf("expected %s staged, staged set:\n%s", res.Path, out)
	}
}

// TestEditFromFileRoundTrip exercises the nvim :w path: read the on-disk file,
// change a field, feed it back through Edit --from-file.
func TestEditFromFileRoundTrip(t *testing.T) {
	root := initGitRepo(t)
	if _, err := Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := Discover(root)
	cfg, _ := s.Config()
	res, err := s.Create(cfg, CreateOpts{Type: item.TypeTicket, Title: "RoundTrip", NoEdit: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	orig, err := os.ReadFile(filepath.Join(root, res.Path))
	if err != nil {
		t.Fatalf("read item file: %v", err)
	}
	edited := strings.Replace(string(orig), `title: "RoundTrip"`, `title: "Round Trip Edited"`, 1)
	editPath := filepath.Join(t.TempDir(), "edit.md")
	if err := os.WriteFile(editPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := s.Edit(cfg, res.Number, EditOpts{FromFile: editPath})
	if err != nil {
		t.Fatalf("Edit --from-file: %v", err)
	}
	if len(r.Changed) != 1 || r.Changed[0] != "title" {
		t.Fatalf("changed = %v, want [title]", r.Changed)
	}
	show, err := s.Show(cfg, res.Number)
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if show.Title != "Round Trip Edited" {
		t.Fatalf("title = %q, want %q", show.Title, "Round Trip Edited")
	}
	// Immutable fields survived the round-trip.
	if show.ID != res.ID || show.Number != res.Number {
		t.Fatalf("identity changed: id=%s number=%s", show.ID, show.Number)
	}
}

// TestEditEditorRepresentsSoftError proves the editor validate-retry loop
// re-presents a soft parse error (a field that parses to a non-nil item but is
// still invalid) instead of silently accepting it — the leniency bug the
// closure previously had. The fake editor injects an unparseable estimate
// first, a valid one second; Edit must loop and land the valid value.
func TestEditEditorRepresentsSoftError(t *testing.T) {
	root := initGitRepo(t)
	if _, err := Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := Discover(root)
	cfg, _ := s.Config()
	res, err := s.Create(cfg, CreateOpts{Type: item.TypeTicket, Title: "SoftErr", NoEdit: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	base, err := os.ReadFile(filepath.Join(root, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	invalid := filepath.Join(dir, "invalid.md")
	valid := filepath.Join(dir, "valid.md")
	counter := filepath.Join(dir, "counter")
	script := filepath.Join(dir, "editor.sh")
	os.WriteFile(invalid, []byte(strings.Replace(string(base), "blocked_by: []", "blocked_by: []\nestimate: notanumber", 1)), 0o644)
	os.WriteFile(valid, []byte(strings.Replace(string(base), "blocked_by: []", "blocked_by: []\nestimate: 5", 1)), 0o644)
	os.WriteFile(script, []byte("#!/bin/sh\n"+
		"n=$(cat \"$KIRA_COUNTER\" 2>/dev/null || echo 0); n=$((n+1)); echo \"$n\" > \"$KIRA_COUNTER\"\n"+
		"if [ \"$n\" -eq 1 ]; then cp \"$KIRA_INVALID\" \"$1\"; else cp \"$KIRA_VALID\" \"$1\"; fi\n"), 0o755)
	t.Setenv("KIRA_COUNTER", counter)
	t.Setenv("KIRA_INVALID", invalid)
	t.Setenv("KIRA_VALID", valid)
	t.Setenv("EDITOR", "sh "+script)

	if _, err := s.Edit(cfg, res.Number, EditOpts{}); err != nil {
		t.Fatalf("Edit (editor): %v", err)
	}
	n, _ := os.ReadFile(counter)
	if strings.TrimSpace(string(n)) != "2" {
		t.Fatalf("editor invoked %s times, want 2 (invalid then valid)", strings.TrimSpace(string(n)))
	}
	show, _ := s.Show(cfg, res.Number)
	if show.Estimate == nil || *show.Estimate != 5 {
		t.Fatalf("estimate = %v, want 5", show.Estimate)
	}
}

// TestListFilters checks the ANDed linear-scan filters and deterministic order.
func TestListFilters(t *testing.T) {
	root := initGitRepo(t)
	if _, err := Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := Discover(root)
	cfg, _ := s.Config()
	mk := func(title, owner string) {
		if _, err := s.Create(cfg, CreateOpts{Type: item.TypeTicket, Title: title, Owner: owner, NoEdit: true}); err != nil {
			t.Fatalf("Create %s: %v", title, err)
		}
	}
	mk("one", "alice")
	mk("two", "bob")
	mk("three", "alice")

	all, err := s.List(cfg, ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if all.Count != 3 {
		t.Fatalf("count = %d, want 3", all.Count)
	}
	if all.Items[0].Number != "KIRA-1" || all.Items[2].Number != "KIRA-3" {
		t.Fatalf("not sorted by number: %v", all.Items)
	}
	byOwner, _ := s.List(cfg, ListOpts{Owner: "alice"})
	if byOwner.Count != 2 {
		t.Fatalf("owner=alice count = %d, want 2", byOwner.Count)
	}
	byCat, _ := s.List(cfg, ListOpts{Category: "todo"})
	if byCat.Count != 3 {
		t.Fatalf("category=todo count = %d, want 3", byCat.Count)
	}
	none, _ := s.List(cfg, ListOpts{State: "DONE"})
	if none.Count != 0 {
		t.Fatalf("state=DONE count = %d, want 0", none.Count)
	}
}
