package core_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func TestCommitPatchShowsLinkedChanges(t *testing.T) {
	s, cfg := newStore(t)
	it := mustCreate(t, s, cfg, "Track code changes")
	repo := gitx.Repo{Dir: s.Root()}
	for name, contents := range map[string]string{
		"code.txt":       "Useful code change\n",
		".gitattributes": "code.txt diff=untrusted\n",
	} {
		if err := os.WriteFile(filepath.Join(s.Root(), name), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := repo.Stage(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.CommitParts("Implement tracked work", "Kira-Ticket: "+it.Number); err != nil {
		t.Fatal(err)
	}
	sha, err := repo.Output("rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"diff.untrusted.textconv", "diff.untrusted.command"} {
		if _, err := repo.Output("config", key, "false"); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GIT_EXTERNAL_DIFF", "false")
	patch, err := s.CommitPatch(context.Background(), cfg, it.ID, sha)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Implement tracked work", "diff --git a/code.txt b/code.txt", "+Useful code change"} {
		if !strings.Contains(patch, want) {
			t.Errorf("patch missing %q: %s", want, patch)
		}
	}
	if strings.Contains(patch, "\x1b[") {
		t.Fatal("patch contains terminal color escapes")
	}
	unrelated := mustCreate(t, s, cfg, "Unrelated")
	if _, err := s.CommitPatch(context.Background(), cfg, unrelated.ID, sha); err == nil {
		t.Fatal("unrelated commit was exposed through an item's preview")
	}
	for _, invalid := range []string{"HEAD", "--output=/tmp/preview", "../code.txt", sha[:8], strings.Repeat("z", 40)} {
		if _, err := s.CommitPatch(context.Background(), cfg, it.ID, invalid); err == nil {
			t.Errorf("invalid SHA %q was accepted", invalid)
		}
	}
	view, err := s.Snapshot(cfg)
	if err != nil {
		t.Fatal(err)
	}
	detail, err := view.Detail(it.ID)
	if err != nil {
		t.Fatal(err)
	}
	unrelatedDetail, err := view.Detail(unrelated.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unrelatedDetail.CommitPatch(context.Background(), sha); err == nil {
		t.Fatal("preloaded detail allowed an unrelated commit")
	}
	for _, invalid := range []string{"HEAD", "--output=/tmp/preview", sha[:8], strings.Repeat("z", 40)} {
		if _, err := detail.CommitPatch(context.Background(), invalid); err == nil {
			t.Errorf("preloaded detail accepted invalid SHA %q", invalid)
		}
	}
	itemPath := storage.New(s.Root()).ItemPath(it.ID)
	if err := os.Rename(itemPath, itemPath+".backup"); err != nil {
		t.Fatal(err)
	}
	loadedPatch, err := detail.CommitPatch(context.Background(), strings.ToUpper(sha))
	if err != nil || loadedPatch != patch {
		t.Fatalf("preloaded detail should preview without rereading its item: %v", err)
	}
}

func TestCommitPatchReferencedCommitAndLimits(t *testing.T) {
	s, cfg := newStore(t)
	it := mustCreate(t, s, cfg, "Mentioned work")
	repo := gitx.Repo{Dir: s.Root()}
	if _, err := repo.Output("commit", "--allow-empty", "-m", "Discuss "+it.Number); err != nil {
		t.Fatal(err)
	}
	sha, err := repo.Output("rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CommitPatch(context.Background(), cfg, it.ID, sha); err != nil {
		t.Fatalf("referenced commit should be viewable: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.CommitPatch(ctx, cfg, it.ID, sha); err == nil {
		t.Fatal("cancelled commit preview succeeded")
	}
	if err := os.WriteFile(filepath.Join(s.Root(), "large.txt"), []byte(strings.Repeat("long changed line\n", 150000)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := repo.Stage("large.txt"); err != nil {
		t.Fatal(err)
	}
	if err := repo.CommitParts("Large diff", "Kira-Ticket: "+it.Number); err != nil {
		t.Fatal(err)
	}
	sha, err = repo.Output("rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	patch, err := s.CommitPatch(context.Background(), cfg, it.ID, sha)
	if err == nil || !strings.Contains(err.Error(), "preview limit") || patch != "" {
		t.Fatalf("oversize patch: %d bytes, %v", len(patch), err)
	}
}
