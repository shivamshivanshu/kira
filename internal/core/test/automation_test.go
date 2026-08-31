package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestUserAutomationFiresWithoutRepoHooks(t *testing.T) {
	s, cfg := newStore(t)
	if len(cfg.Automation) != 0 {
		t.Fatalf("precondition: repo automation must be empty, got %d", len(cfg.Automation))
	}
	sentinel := filepath.Join(t.TempDir(), "fired")
	cfg.UserAutomation = []datamodel.AutomationHook{{
		Name: "usertier",
		On:   datamodel.EventItemCreated,
		Run:  "touch " + sentinel,
	}}
	if _, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "hooked", NoEdit: true}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("user-tier hook did not fire (sentinel missing): %v", err)
	}
}

// The grant must land outside the worktree, or a hostile repo could commit its
// own and have its hooks run on clone (KIRA-108).
func TestAutomationTrustGrantIsNotCommittable(t *testing.T) {
	s, cfg := newStore(t)
	cfg.Automation = []datamodel.AutomationHook{{
		Name: "repohook",
		On:   datamodel.EventItemCreated,
		Run:  "true",
	}}

	if _, err := s.AutomationTrust(cfg); err != nil {
		t.Fatalf("AutomationTrust: %v", err)
	}

	if _, err := os.Stat(filepath.Join(s.Root(), ".kira", ".cache", "automation.trust")); !os.IsNotExist(err) {
		t.Fatal("the grant must not be written inside the worktree")
	}
	if !s.AutomationList(cfg).Trusted {
		t.Fatal("the grant must still be readable after being written outside the worktree")
	}
}

func TestUntrustedRepoHookDoesNotFire(t *testing.T) {
	s, cfg := newStore(t)
	sentinel := filepath.Join(t.TempDir(), "fired")
	cfg.Automation = []datamodel.AutomationHook{{
		Name: "repohook",
		On:   datamodel.EventItemCreated,
		Run:  "touch " + sentinel,
	}}

	if _, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "untrusted", NoEdit: true}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := os.Stat(sentinel); err == nil {
		t.Fatal("an untrusted repo hook must not run")
	}
}
