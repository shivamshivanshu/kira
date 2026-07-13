package automation_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/automation"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestUserHooksExcludedFromTrustHash(t *testing.T) {
	repoOnly := &datamodel.Config{Automation: []datamodel.AutomationHook{
		{Name: "repo", On: datamodel.EventItemCreated, Run: "true"},
	}}
	withUser := &datamodel.Config{
		Automation:     repoOnly.Automation,
		UserAutomation: []datamodel.AutomationHook{{Name: "user", On: datamodel.EventItemCreated, Run: "true"}},
	}
	if automation.Hash(repoOnly) != automation.Hash(withUser) {
		t.Fatal("adding user hooks must not change the repo trust hash")
	}
}

func TestUserHookEditDoesNotRevokeRepoTrust(t *testing.T) {
	dir := t.TempDir()
	cfg := &datamodel.Config{
		Automation:     []datamodel.AutomationHook{{Name: "repo", On: datamodel.EventItemCreated, Run: "true"}},
		UserAutomation: []datamodel.AutomationHook{{Name: "user", On: datamodel.EventItemCreated, Run: "echo one"}},
	}
	if _, err := automation.Grant(dir, cfg); err != nil {
		t.Fatalf("grant: %v", err)
	}
	if !automation.Trusted(dir, cfg) {
		t.Fatal("granted repo config must be trusted")
	}

	cfg.UserAutomation[0].Run = "echo two"
	cfg.UserAutomation = append(cfg.UserAutomation, datamodel.AutomationHook{Name: "user2", On: datamodel.EventItemCreated, Run: "true"})
	if !automation.Trusted(dir, cfg) {
		t.Fatal("editing user hooks must not revoke repo trust")
	}

	cfg.Automation[0].Run = "false"
	if automation.Trusted(dir, cfg) {
		t.Fatal("editing a repo hook must revoke trust")
	}
}
