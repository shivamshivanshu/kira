package core

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
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
	if err := s.prepareTree(config.Default(), repo, SyncOpts{}, &syncx.Report{}); err != nil {
		t.Fatalf("prepareTree clean: %v", err)
	}
}

func TestPrepareTreeCommitsDirty(t *testing.T) {
	s, repo := seededSyncRepo(t)
	dirtyKira(t, s, "IN_PROGRESS")

	if err := s.prepareTree(config.Default(), repo, SyncOpts{Dirty: syncx.DirtyCommit}, &syncx.Report{}); err != nil {
		t.Fatalf("prepareTree commit: %v", err)
	}
	dirty, _ := repo.DirtyPaths(".kira")
	if len(dirty) != 0 {
		t.Fatalf("commit left dirty paths: %v", dirty)
	}
}

func TestPrepareTreeRefusesDirtyWithoutCommit(t *testing.T) {
	s, repo := seededSyncRepo(t)
	dirtyKira(t, s, "IN_PROGRESS")

	err := s.prepareTree(config.Default(), repo, SyncOpts{}, &syncx.Report{})
	if err == nil || !strings.Contains(err.Error(), "re-run with --commit") {
		t.Fatalf("err = %v, want refusal on uncommitted changes even under commit.mode auto", err)
	}
}
