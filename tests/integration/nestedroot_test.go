package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/syncx"
)

func nestedStoreRoot(t *testing.T, gitRoot string) string {
	t.Helper()
	sub := filepath.Join(gitRoot, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	initStore(t, sub)
	return sub
}

func writeNestedItem(t *testing.T, storeRoot string, it *datamodel.Item) {
	t.Helper()
	abs := filepath.Join(storeRoot, filepath.FromSlash(ticketRel(it.ID)))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(codec.Serialize(it)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNestedRootKiraCommitSummarizesItems(t *testing.T) {
	t.Parallel()
	gitRoot := initGitRepo(t)
	sub := nestedStoreRoot(t, gitRoot)
	s, cfg := discoverStore(t, sub)

	res, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "Nested", NoEdit: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	abs := filepath.Join(sub, filepath.FromSlash(res.Path))
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), `title: "Nested"`, `title: "Nested edited"`, 1)
	if edited == string(data) {
		t.Fatal("item file has no title line to edit")
	}
	if err := os.WriteFile(abs, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	cr, err := s.CommitKira(cfg)
	if err != nil {
		t.Fatalf("CommitKira: %v", err)
	}
	if len(cr.Items) != 1 || cr.Items[0] != res.Number {
		t.Fatalf("items = %v, want [%s]", cr.Items, res.Number)
	}
	if !strings.Contains(cr.Subject, "1 items") {
		t.Fatalf("subject = %q, want item-count subject", cr.Subject)
	}
	dirty, err := (gitx.Repo{Dir: gitRoot}).Output("status", "--porcelain", "--", "sub/.kira")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if dirty != "" {
		t.Fatalf("kira tree still dirty after commit:\n%s", dirty)
	}
}

func TestNestedRootSyncDirtyCommitAutoResolves(t *testing.T) {
	t.Parallel()
	bare := bareRemote(t)

	seed := cloneWorld(t, bare)
	seedSub := nestedStoreRoot(t, seed.root)
	writeNestedItem(t, seedSub, matrixItem(ulidX, "KIRA-1", "X"))
	seed.commit("base")
	if _, err := seed.repo.Output("push", "-u", "origin", "main"); err != nil {
		t.Fatalf("push seed: %v", err)
	}

	ours := cloneWorld(t, bare)
	them := cloneWorld(t, bare)

	theirs := matrixItem(ulidX, "KIRA-1", "X")
	theirs.State, theirs.Updated = "DONE", tsEarly
	writeNestedItem(t, filepath.Join(them.root, "sub"), theirs)
	them.commit("theirs")
	if _, err := them.repo.Output("push", "origin", "main"); err != nil {
		t.Fatalf("push theirs: %v", err)
	}

	mine := matrixItem(ulidX, "KIRA-1", "X")
	mine.State, mine.Updated = "REVIEW", tsLate
	oursSub := filepath.Join(ours.root, "sub")
	writeNestedItem(t, oursSub, mine)

	s, cfg := discoverStore(t, oursSub)
	report, err := s.Sync(cfg, core.SyncOpts{Dirty: syncx.DirtyCommit}, nil)
	if err != nil {
		t.Fatalf("Sync: %v (report %+v)", err, report)
	}
	prepared := false
	for _, step := range report.Steps {
		if step.Name == "prepare" && step.Status == syncx.StepDone && strings.Contains(step.Detail, "committed") {
			prepared = true
		}
	}
	if !prepared {
		t.Fatalf("no dirty-commit prepare step in report: %+v", report)
	}

	raw, err := os.ReadFile(filepath.Join(oursSub, filepath.FromSlash(ticketRel(ulidX))))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "<<<<<<<") {
		t.Fatalf("sync left conflict markers:\n%s", raw)
	}
	merged, err := codec.Parse(string(raw))
	if err != nil {
		t.Fatalf("merged item unparseable: %v", err)
	}
	if merged.State != "REVIEW" {
		t.Fatalf("state = %s, want REVIEW (later-updated side wins)", merged.State)
	}
	assertNoUnmerged(t, ours.repo)
}

func TestNestedRootPreCommitValidatesStagedItems(t *testing.T) {
	t.Parallel()
	gitRoot := initGitRepo(t)
	sub := nestedStoreRoot(t, gitRoot)
	s, cfg := discoverStore(t, sub)

	bad := filepath.Join(sub, ".kira", "tickets", "broken.md")
	if err := os.WriteFile(bad, []byte("not a kira item\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (gitx.Repo{Dir: gitRoot}).Output("add", "--", "sub/.kira/tickets"); err != nil {
		t.Fatalf("git add: %v", err)
	}

	err := s.ValidateStaged(cfg)
	if err == nil {
		t.Fatal("ValidateStaged must reject an unparseable staged item in a nested store")
	}
	if !strings.Contains(err.Error(), "broken.md") {
		t.Fatalf("error %q does not name the offending file", err)
	}
}

func TestNestedRootPostMergeHookReindexes(t *testing.T) {
	t.Parallel()
	gitRoot := initGitRepo(t)
	sub := nestedStoreRoot(t, gitRoot)
	s, cfg := discoverStore(t, sub)

	if _, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "Indexed", NoEdit: true}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	writeNestedItem(t, sub, matrixItem(ulidY, "KIRA-2", "Hand-written"))
	repo := gitx.Repo{Dir: gitRoot}
	if _, err := repo.Output("add", "-A"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if _, err := repo.Output("commit", "-m", "hand-written ticket"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	hook := exec.Command("kira", "hooks", "run", "post-merge")
	hook.Dir = sub
	if out, err := hook.CombinedOutput(); err != nil {
		t.Fatalf("post-merge hook: %v\n%s", err, out)
	}

	res, err := s.Index(cfg, false, false)
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if res.Action != "fresh" {
		t.Fatalf("index after hook = %s (%s), want fresh", res.Action, res.Reason)
	}
	if res.Items != 2 {
		t.Fatalf("indexed items = %d, want 2", res.Items)
	}
}
