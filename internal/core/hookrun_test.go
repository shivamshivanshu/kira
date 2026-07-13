package core

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func stagedFixture(t *testing.T) (*Store, *datamodel.Config, gitx.Repo) {
	t.Helper()
	dir := t.TempDir()
	if err := testutil.GitInit(dir); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := Init(dir, "KIRA", false); err != nil {
		t.Fatalf("init store: %v", err)
	}
	s, err := Discover(dir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	cfg, err := s.Config()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	return s, cfg, gitx.Repo{Dir: dir}
}

func stageItem(t *testing.T, s *Store, repo gitx.Repo, it *datamodel.Item) {
	t.Helper()
	if _, err := s.fs().WriteItem(it); err != nil {
		t.Fatalf("write item: %v", err)
	}
	if err := repo.Stage(".kira"); err != nil {
		t.Fatalf("stage: %v", err)
	}
}

func TestValidateStagedAcceptsValidItem(t *testing.T) {
	s, cfg, repo := stagedFixture(t)
	stageItem(t, s, repo, eventTicket())

	if err := s.ValidateStaged(cfg); err != nil {
		t.Fatalf("valid staged item rejected: %v", err)
	}
}

func TestValidateStagedRejectsInvalidItem(t *testing.T) {
	s, cfg, repo := stagedFixture(t)
	bad := eventTicket()
	bad.State = "BOGUS"
	stageItem(t, s, repo, bad)

	if err := s.ValidateStaged(cfg); err == nil {
		t.Fatal("staged item with unknown state must be rejected")
	}
}
