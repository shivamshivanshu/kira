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
