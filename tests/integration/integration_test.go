package integration

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func initGitRepo(t *testing.T) string {
	return testutil.InitGitRepo(t)
}

func commitCount(t *testing.T, root string) int {
	t.Helper()
	out, err := gitx.Repo{Dir: root}.Output("rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("rev-list: %v", err)
	}
	n, err := strconv.Atoi(out)
	if err != nil {
		t.Fatalf("parse count %q: %v", out, err)
	}
	return n
}

func TestAutoModeOneCommitPerMutation(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if got := commitCount(t, root); got != 1 {
		t.Fatalf("after init: %d commits, want 1", got)
	}
	s, err := core.Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}

	res, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "First", NoEdit: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Number != "KIRA-1" {
		t.Fatalf("number = %q, want KIRA-1", res.Number)
	}
	if got := commitCount(t, root); got != 2 {
		t.Fatalf("after create: %d commits, want 2", got)
	}

	if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "state", Value: "IN_PROGRESS"}}}); err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if got := commitCount(t, root); got != 3 {
		t.Fatalf("after edit: %d commits, want 3", got)
	}

	if r, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "state", Value: "IN_PROGRESS"}}}); err != nil {
		t.Fatalf("no-op Edit: %v", err)
	} else if len(r.Changed) != 0 {
		t.Fatalf("no-op edit reported changes: %v", r.Changed)
	}
	if got := commitCount(t, root); got != 3 {
		t.Fatalf("after no-op edit: %d commits, want 3", got)
	}
}

func TestManualModeStagesNoCommit(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	repo := gitx.Repo{Dir: root}
	cfgPath := filepath.Join(root, ".kira", "config.yaml")
	data, _ := os.ReadFile(cfgPath)
	_ = os.WriteFile(cfgPath, []byte(strings.Replace(string(data), "mode: auto", "mode: manual", 1)), 0o644)
	for _, args := range [][]string{
		{"add", ".kira/config.yaml"},
		{"commit", "-m", "manual mode"},
	} {
		if _, err := repo.Output(args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	before := commitCount(t, root)

	s, _ := core.Discover(root)
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if cfg.Commit.Mode != datamodel.CommitManual {
		t.Fatalf("mode = %q, want manual", cfg.Commit.Mode)
	}
	res, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "Staged", NoEdit: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got := commitCount(t, root); got != before {
		t.Fatalf("manual mode committed: count %d, want %d", got, before)
	}
	staged, err := repo.Output("diff", "--cached", "--name-only")
	if err != nil {
		t.Fatalf("diff --cached: %v", err)
	}
	if !strings.Contains(staged, res.Path) {
		t.Fatalf("expected %s staged, staged set:\n%s", res.Path, staged)
	}
}

func TestEditFromFileRoundTrip(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := core.Discover(root)
	cfg, _ := s.Config()
	res, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "RoundTrip", NoEdit: true})
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

	r, err := s.Edit(cfg, res.Number, core.EditOpts{FromFile: editPath})
	if err != nil {
		t.Fatalf("Edit --from-file: %v", err)
	}
	if len(r.Changed) != 1 || r.Changed[0] != "title" {
		t.Fatalf("changed = %v, want [title]", r.Changed)
	}
	show, err := s.Show(cfg, res.Number, "")
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if show.Title != "Round Trip Edited" {
		t.Fatalf("title = %q, want %q", show.Title, "Round Trip Edited")
	}
	if show.ID != res.ID || show.Number != res.Number {
		t.Fatalf("identity changed: id=%s number=%s", show.ID, show.Number)
	}
}

func TestEditEditorRepresentsSoftError(t *testing.T) {
	root := initGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := core.Discover(root)
	cfg, _ := s.Config()
	res, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "SoftErr", NoEdit: true})
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
	_ = os.WriteFile(invalid, []byte(strings.Replace(string(base), "blocked_by: []", "blocked_by: []\nestimate: notanumber", 1)), 0o644)
	_ = os.WriteFile(valid, []byte(strings.Replace(string(base), "blocked_by: []", "blocked_by: []\nestimate: 5", 1)), 0o644)
	_ = os.WriteFile(script, []byte("#!/bin/sh\n"+
		"n=$(cat \"$KIRA_COUNTER\" 2>/dev/null || echo 0); n=$((n+1)); echo \"$n\" > \"$KIRA_COUNTER\"\n"+
		"if [ \"$n\" -eq 1 ]; then cp \"$KIRA_INVALID\" \"$1\"; else cp \"$KIRA_VALID\" \"$1\"; fi\n"), 0o755)
	t.Setenv("KIRA_COUNTER", counter)
	t.Setenv("KIRA_INVALID", invalid)
	t.Setenv("KIRA_VALID", valid)
	t.Setenv("EDITOR", "sh "+script)

	if _, err := s.Edit(cfg, res.Number, core.EditOpts{}); err != nil {
		t.Fatalf("Edit (editor): %v", err)
	}
	n, _ := os.ReadFile(counter)
	if strings.TrimSpace(string(n)) != "2" {
		t.Fatalf("editor invoked %s times, want 2 (invalid then valid)", strings.TrimSpace(string(n)))
	}
	show, _ := s.Show(cfg, res.Number, "")
	if show.Estimate == nil || *show.Estimate != 5 {
		t.Fatalf("estimate = %v, want 5", show.Estimate)
	}
}

func TestListFilters(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t)
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
	s, _ := core.Discover(root)
	cfg, _ := s.Config()
	mk := func(title, owner string) {
		if _, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: title, Owner: owner, NoEdit: true}); err != nil {
			t.Fatalf("Create %s: %v", title, err)
		}
	}
	mk("one", "alice")
	mk("two", "bob")
	mk("three", "alice")

	all, err := s.List(cfg, core.ListOpts{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if all.Count != 3 {
		t.Fatalf("count = %d, want 3", all.Count)
	}
	if all.Items[0].Number != "KIRA-1" || all.Items[2].Number != "KIRA-3" {
		t.Fatalf("not sorted by number: %v", all.Items)
	}
	byOwner, _ := s.List(cfg, core.ListOpts{Owner: "alice"})
	if byOwner.Count != 2 {
		t.Fatalf("owner=alice count = %d, want 2", byOwner.Count)
	}
	byCat, _ := s.List(cfg, core.ListOpts{Category: "todo"})
	if byCat.Count != 3 {
		t.Fatalf("category=todo count = %d, want 3", byCat.Count)
	}
	none, _ := s.List(cfg, core.ListOpts{State: "DONE"})
	if none.Count != 0 {
		t.Fatalf("state=DONE count = %d, want 0", none.Count)
	}
}
