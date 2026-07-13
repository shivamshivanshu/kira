package automation

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestMatchedHooksFiltersByEnabledAndEvent(t *testing.T) {
	disabled := false
	cfg := &datamodel.Config{Automation: []datamodel.AutomationHook{
		{Name: "enabled-match", On: datamodel.EventItemCreated, Run: "true"},
		{Name: "disabled", On: datamodel.EventItemCreated, Run: "true", Enabled: &disabled},
		{Name: "other-event", On: datamodel.EventItemStateChanged, Run: "true"},
	}}
	got := matchedHooks(cfg, Event{Name: datamodel.EventItemCreated})
	if len(got) != 1 || got[0].Name != "enabled-match" {
		t.Fatalf("matchedHooks = %+v, want only the enabled hook on the fired event", got)
	}
}
