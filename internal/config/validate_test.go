package config

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestValidateAutomationHooksMatchKeysPerEvent(t *testing.T) {
	cases := []struct {
		name    string
		on      datamodel.EventName
		match   datamodel.AutomationMatch
		wantErr bool
	}{
		{"created: type is populated", datamodel.EventItemCreated, datamodel.AutomationMatch{Type: "ticket"}, false},
		{"created: to is never populated", datamodel.EventItemCreated, datamodel.AutomationMatch{To: "done"}, true},
		{"created: from is never populated", datamodel.EventItemCreated, datamodel.AutomationMatch{From: "todo"}, true},
		{"state_changed: to is populated", datamodel.EventItemStateChanged, datamodel.AutomationMatch{To: "done"}, false},
		{"state_changed: from is populated", datamodel.EventItemStateChanged, datamodel.AutomationMatch{From: "todo"}, false},
		{"state_changed: type is populated", datamodel.EventItemStateChanged, datamodel.AutomationMatch{Type: "ticket"}, false},
		{"sync_completed: type is never populated", datamodel.EventSyncCompleted, datamodel.AutomationMatch{Type: "ticket"}, true},
		{"sync_completed: to is never populated", datamodel.EventSyncCompleted, datamodel.AutomationMatch{To: "done"}, true},
		{"sync_completed: from is never populated", datamodel.EventSyncCompleted, datamodel.AutomationMatch{From: "todo"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hooks := []datamodel.AutomationHook{{On: tc.on, Run: "true", Match: &tc.match}}
			err := validateAutomationHooks("automation", hooks)
			if tc.wantErr && err == nil {
				t.Fatalf("validateAutomationHooks(%+v) = nil, want error for a dead match key", tc.match)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("validateAutomationHooks(%+v) = %v, want nil", tc.match, err)
			}
		})
	}
}
