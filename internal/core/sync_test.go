package core

import (
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
