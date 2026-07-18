package integration

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func setMergePolicyManual(t *testing.T, root string) {
	t.Helper()
	p := filepath.Join(root, ".kira", "config.yaml")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	out := strings.Replace(string(data), "policy: auto", "policy: manual", 1)
	if out == string(data) {
		t.Fatal("config has no `policy: auto` to flip to manual")
	}
	if err := os.WriteFile(p, []byte(out), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func initStore(t *testing.T, root string) {
	t.Helper()
	if _, err := core.Init(root, "KIRA", false); err != nil {
		t.Fatalf("Init: %v", err)
	}
}

func coreDiscover(root string) (*core.Store, error) { return core.Discover(root) }

func TestMain(m *testing.M) {
	testutil.ApplyHermeticEnvironment()
	bin := testutil.KiraBinary(nil)
	_ = os.Setenv("PATH", filepath.Dir(bin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	code := m.Run()
	os.Exit(code)
}

const mergeULID = "01J8X8Q7RZTN5Y3VXW2A9K4E7F"

func mergeRelPath() string { return ".kira/tickets/" + mergeULID + ".md" }

func mergeItem(mut func(*datamodel.Item)) *datamodel.Item {
	it := &datamodel.Item{
		ID:        mergeULID,
		Number:    "KIRA-1",
		Aliases:   []string{},
		Type:      datamodel.TypeTicket,
		Title:     "Merge subject",
		State:     "TODO",
		Labels:    []string{},
		Epic:      nil,
		BlockedBy: []string{},
		Created:   "2026-01-01T00:00:00Z",
		Updated:   "2026-01-01T00:00:00Z",
		Body:      "## Description\n\nbody\n",
	}
	if mut != nil {
		mut(it)
	}
	return it
}

func writeCommit(t *testing.T, repo gitx.Repo, root string, it *datamodel.Item, msg string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(mergeRelPath()))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(codec.Serialize(it)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Output("add", "--", mergeRelPath()); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := repo.Output("commit", "-m", msg); err != nil {
		t.Fatalf("git commit: %v", err)
	}
}

func gitMerger(b, o, tt string) (string, bool) {
	mm, c, err := gitx.MergeText(b, o, tt)
	if err != nil {
		return "", true
	}
	return mm, c
}

// diverge commits base on the current branch, then applies ours on it and
// theirs on a sibling branch, leaving the working tree on the original branch.
func diverge(t *testing.T, repo gitx.Repo, root string, base, ours, theirs *datamodel.Item) (mainBranch string) {
	t.Helper()
	mainBranch, err := repo.Output("branch", "--show-current")
	if err != nil {
		t.Fatalf("branch --show-current: %v", err)
	}
	writeCommit(t, repo, root, base, "base item")
	if _, err := repo.Output("checkout", "-b", "other"); err != nil {
		t.Fatalf("checkout -b other: %v", err)
	}
	writeCommit(t, repo, root, theirs, "theirs change")
	if _, err := repo.Output("checkout", mainBranch); err != nil {
		t.Fatalf("checkout %s: %v", mainBranch, err)
	}
	writeCommit(t, repo, root, ours, "ours change")
	return mainBranch
}

func registerDriver(t *testing.T, root string) {
	t.Helper()
	s, err := coreDiscover(root)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if err := s.RegisterMergeDriver(); err != nil {
		t.Fatalf("register driver: %v", err)
	}
}

func TestResolveCleansPlainMergeConflict(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t)
	initStore(t, root)
	repo := gitx.Repo{Dir: root}

	base := mergeItem(nil)
	ours := mergeItem(func(it *datamodel.Item) { it.State = "REVIEW"; it.Updated = "2026-03-01T00:00:00Z" })
	theirs := mergeItem(func(it *datamodel.Item) { it.State = "DONE"; it.Updated = "2026-03-02T00:00:00Z" })
	diverge(t, repo, root, base, ours, theirs)

	if _, err := repo.Output("merge", "other"); err == nil {
		t.Fatal("plain merge without driver should conflict")
	}
	abs := filepath.Join(root, filepath.FromSlash(mergeRelPath()))
	raw, _ := os.ReadFile(abs)
	if !strings.Contains(string(raw), "<<<<<<<") {
		t.Fatalf("expected conflict markers before resolve:\n%s", raw)
	}

	s, err := coreDiscover(root)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	res, err := s.Resolve(cfg, nil, false)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(res.Resolved) != 1 || res.Resolved[0].Number != "KIRA-1" {
		t.Fatalf("resolved = %+v, want KIRA-1", res.Resolved)
	}
	clean, _ := os.ReadFile(abs)
	if strings.Contains(string(clean), "<<<<<<<") {
		t.Fatalf("resolve left markers:\n%s", clean)
	}
	merged, err := codec.Parse(string(clean))
	if err != nil {
		t.Fatalf("resolved file unparseable: %v", err)
	}
	if merged.State != "DONE" {
		t.Fatalf("resolved state = %q, want DONE (theirs updated later)", merged.State)
	}
	if unmerged, _ := repo.Output("ls-files", "-u"); unmerged != "" {
		t.Fatalf("path still unmerged after resolve: %s", unmerged)
	}
	staged, _ := repo.Output("diff", "--cached", "--name-only")
	if !strings.Contains(staged, mergeRelPath()) {
		t.Fatalf("resolved file not staged: %s", staged)
	}
}

func writeScratch(t *testing.T, root, name, content string) string {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMergeFileManualPolicyConflictExit(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t)
	initStore(t, root)
	setMergePolicyManual(t, root)
	repo := gitx.Repo{Dir: root}

	base := writeScratch(t, root, "mf_base", "line1\nMID\nline3\n")
	ours := writeScratch(t, root, "mf_ours", "line1\nOURS\nline3\n")
	theirs := writeScratch(t, root, "mf_theirs", "line1\nTHEIRS\nline3\n")

	_, err := core.MergeFile(repo, base, ours, theirs)
	if err == nil {
		t.Fatal("manual policy with an overlapping edit must return a conflict error")
	}
	var ce *errx.Error
	if !errors.As(err, &ce) || ce.Code != errx.ExitConflict {
		t.Fatalf("want conflict exit code %d, got err=%v", errx.ExitConflict, err)
	}
	got, _ := os.ReadFile(ours)
	if !strings.Contains(string(got), "<<<<<<<") {
		t.Fatalf("manual policy must leave conflict markers in ours:\n%s", got)
	}
}

func TestMergeFileManualPolicyCleanTextMerge(t *testing.T) {
	t.Parallel()
	root := initGitRepo(t)
	initStore(t, root)
	setMergePolicyManual(t, root)
	repo := gitx.Repo{Dir: root}

	base := writeScratch(t, root, "mf_base", "a\nb\nc\n")
	ours := writeScratch(t, root, "mf_ours", "OURS\nb\nc\n")
	theirs := writeScratch(t, root, "mf_theirs", "a\nb\nTHEIRS\n")

	if _, err := core.MergeFile(repo, base, ours, theirs); err != nil {
		t.Fatalf("non-overlapping manual merge must succeed cleanly: %v", err)
	}
	got, _ := os.ReadFile(ours)
	s := string(got)
	if strings.Contains(s, "<<<<<<<") {
		t.Fatalf("clean text merge left markers:\n%s", s)
	}
	if !strings.Contains(s, "OURS") || !strings.Contains(s, "THEIRS") {
		t.Fatalf("clean text merge should union both sides:\n%s", s)
	}
}
