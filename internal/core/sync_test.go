package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/syncx"
)

func seededSyncRepo(t *testing.T) (*Store, gitx.Repo) {
	t.Helper()
	s := eventRepo(t)
	commitState(t, s, eventTicket(), "TODO", "2026-02-01")
	return s, gitx.Repo{Dir: s.root}
}

func dirtyKira(t *testing.T, s *Store, state string) {
	t.Helper()
	it := eventTicket()
	it.State = state
	if _, err := s.fs().WriteItem(it); err != nil {
		t.Fatalf("write dirty item: %v", err)
	}
}

func TestPrepareTreeCleanTree(t *testing.T) {
	s, repo := seededSyncRepo(t)
	stashed, err := s.prepareTree(config.Default(), repo, SyncOpts{}, &syncx.Report{})
	if err != nil {
		t.Fatalf("prepareTree clean: %v", err)
	}
	if stashed {
		t.Fatal("clean tree must not stash")
	}
}

func TestPrepareTreeStashesDirty(t *testing.T) {
	s, repo := seededSyncRepo(t)
	dirtyKira(t, s, "IN_PROGRESS")

	stashed, err := s.prepareTree(config.Default(), repo, SyncOpts{Dirty: syncx.DirtyStash}, &syncx.Report{})
	if err != nil {
		t.Fatalf("prepareTree stash: %v", err)
	}
	if !stashed {
		t.Fatal("dirty tree with stash policy must report stashed")
	}
	dirty, _ := repo.DirtyPaths(".kira")
	if len(dirty) != 0 {
		t.Fatalf("stash left dirty paths: %v", dirty)
	}
}

func TestPrepareTreeCommitsDirty(t *testing.T) {
	s, repo := seededSyncRepo(t)
	dirtyKira(t, s, "IN_PROGRESS")

	stashed, err := s.prepareTree(config.Default(), repo, SyncOpts{Dirty: syncx.DirtyCommit}, &syncx.Report{})
	if err != nil {
		t.Fatalf("prepareTree commit: %v", err)
	}
	if stashed {
		t.Fatal("commit policy must not stash")
	}
	dirty, _ := repo.DirtyPaths(".kira")
	if len(dirty) != 0 {
		t.Fatalf("commit left dirty paths: %v", dirty)
	}
}

func TestPrepareTreeManualCommitRefusesDirty(t *testing.T) {
	s, repo := seededSyncRepo(t)
	dirtyKira(t, s, "IN_PROGRESS")

	cfg := config.Default()
	cfg.Commit.Mode = datamodel.CommitManual
	_, err := s.prepareTree(cfg, repo, SyncOpts{Dirty: syncx.DirtyAuto}, &syncx.Report{})
	if err == nil || !strings.Contains(err.Error(), "uncommitted kira changes") {
		t.Fatalf("err = %v, want refusal on uncommitted changes under manual commit mode", err)
	}
}

func TestPopStashRestoresCleanly(t *testing.T) {
	s, repo := seededSyncRepo(t)
	dirtyKira(t, s, "IN_PROGRESS")
	if err := repo.Stash(); err != nil {
		t.Fatalf("stash: %v", err)
	}

	if err := s.popStash(config.Default(), repo, &syncx.Report{}); err != nil {
		t.Fatalf("popStash clean: %v", err)
	}
	dirty, _ := repo.DirtyPaths(".kira")
	if len(dirty) == 0 {
		t.Fatal("popStash must restore the stashed edit to the working tree")
	}
}

func conflictingStash(t *testing.T) (*Store, gitx.Repo) {
	t.Helper()
	s, repo := seededSyncRepo(t)
	dirtyKira(t, s, "IN_PROGRESS")
	if err := repo.Stash(); err != nil {
		t.Fatalf("stash: %v", err)
	}
	commitState(t, s, eventTicket(), "REVIEW", "2026-02-02")
	return s, repo
}

func TestPopStashAutoResolvesAndCleansUpStash(t *testing.T) {
	s, repo := conflictingStash(t)

	if err := s.popStash(config.Default(), repo, &syncx.Report{}); err != nil {
		t.Fatalf("popStash auto-resolve: %v", err)
	}

	staged, err := repo.StagedPaths()
	if err != nil {
		t.Fatalf("staged paths: %v", err)
	}
	if len(staged) != 0 {
		t.Errorf("auto-resolved pop left staged paths: %v", staged)
	}
	stashes, err := repo.Output("stash", "list")
	if err != nil {
		t.Fatalf("stash list: %v", err)
	}
	if stashes != "" {
		t.Errorf("auto-resolved pop left a stash entry: %q", stashes)
	}
}

func TestPopStashAutoResolveUnstagesCleanDeletion(t *testing.T) {
	s, repo := seededSyncRepo(t)
	extraPath := filepath.Join(s.root, "extra.txt")
	if err := os.WriteFile(extraPath, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, s, "2026-02-01T11:00:00Z", "add", "extra.txt")
	gitRun(t, s, "2026-02-01T11:00:00Z", "commit", "-m", "add extra")

	dirtyKira(t, s, "IN_PROGRESS")
	if err := os.Remove(extraPath); err != nil {
		t.Fatal(err)
	}
	if err := repo.Stash(); err != nil {
		t.Fatalf("stash: %v", err)
	}
	commitState(t, s, eventTicket(), "REVIEW", "2026-02-02")

	if err := s.popStash(config.Default(), repo, &syncx.Report{}); err != nil {
		t.Fatalf("popStash auto-resolve: %v", err)
	}

	// Not StagedPaths: it filters to ACM and would miss a staged deletion.
	staged, err := repo.Output("diff", "--cached", "--name-only")
	if err != nil {
		t.Fatalf("diff --cached: %v", err)
	}
	if staged != "" {
		t.Errorf("clean-merged deletion left staged: %q", staged)
	}
	if _, err := os.Stat(extraPath); !os.IsNotExist(err) {
		t.Errorf("extra.txt still exists on disk, want it deleted")
	}
}

func TestPopStashManualConflictSurfaces(t *testing.T) {
	s, repo := seededSyncRepo(t)
	dirtyKira(t, s, "IN_PROGRESS")
	if err := repo.Stash(); err != nil {
		t.Fatalf("stash: %v", err)
	}
	commitState(t, s, eventTicket(), "REVIEW", "2026-02-02")

	cfg := config.Default()
	cfg.Merge.Policy = datamodel.MergeManual
	err := s.popStash(cfg, repo, &syncx.Report{})
	if err == nil || !strings.Contains(err.Error(), "stash pop") {
		t.Fatalf("err = %v, want manual-policy stash-pop conflict surfaced", err)
	}
}
