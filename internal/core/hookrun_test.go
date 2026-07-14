package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func stagedFixture(t *testing.T) (*Store, *datamodel.Config, gitx.Repo) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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

func TestPrepareCommitMsgHookNoTicketBranchLeavesMsgUntouched(t *testing.T) {
	s, _, repo := stagedFixture(t)
	if err := repo.CheckoutNew("plain-branch"); err != nil {
		t.Fatalf("checkout: %v", err)
	}
	msg := filepath.Join(t.TempDir(), "COMMIT_MSG")
	if err := os.WriteFile(msg, []byte("wip: something\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.PrepareCommitMsgHook(msg); err != nil {
		t.Fatalf("PrepareCommitMsgHook: %v", err)
	}
	got, err := os.ReadFile(msg)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "wip: something\n" {
		t.Errorf("message rewritten on a non-ticket branch: %q", got)
	}
}

func TestPrepareCommitMsgHookTicketBranchAddsTrailer(t *testing.T) {
	s, cfg, repo := stagedFixture(t)
	stageItem(t, s, repo, eventTicket())
	if _, err := s.CommitKira(cfg); err != nil {
		t.Fatalf("commit fixture: %v", err)
	}
	if err := repo.CheckoutNew("kira/kira-1-t"); err != nil {
		t.Fatalf("checkout: %v", err)
	}
	msg := filepath.Join(t.TempDir(), "COMMIT_MSG")
	if err := os.WriteFile(msg, []byte("fix: the bug\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.PrepareCommitMsgHook(msg); err != nil {
		t.Fatalf("PrepareCommitMsgHook: %v", err)
	}
	got, err := os.ReadFile(msg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Kira-Ticket: KIRA-1") {
		t.Errorf("trailer missing on ticket branch: %q", got)
	}
}
