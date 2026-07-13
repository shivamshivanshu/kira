package automation

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestMatchedHooksFiltersByEnabledAndEvent(t *testing.T) {
	disabled := false
	hooks := []datamodel.AutomationHook{
		{Name: "enabled-match", On: datamodel.EventItemCreated, Run: "true"},
		{Name: "disabled", On: datamodel.EventItemCreated, Run: "true", Enabled: &disabled},
		{Name: "other-event", On: datamodel.EventItemStateChanged, Run: "true"},
	}
	got := matchedHooks(hooks, Event{Name: datamodel.EventItemCreated})
	if len(got) != 1 || got[0].Name != "enabled-match" {
		t.Fatalf("matchedHooks = %+v, want only the enabled hook on the fired event", got)
	}
}

func TestFiringSetTrustPartition(t *testing.T) {
	repo := []datamodel.AutomationHook{{Name: "repo"}}
	user := []datamodel.AutomationHook{{Name: "user"}}

	untrusted := firingSet(false, repo, user)
	if len(untrusted) != 1 || untrusted[0].Name != "user" {
		t.Fatalf("untrusted firing set = %+v, want user hooks only", untrusted)
	}

	trusted := firingSet(true, repo, user)
	if len(trusted) != 2 || trusted[0].Name != "repo" || trusted[1].Name != "user" {
		t.Fatalf("trusted firing set = %+v, want repo then user", trusted)
	}
}
