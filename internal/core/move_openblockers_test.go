package core

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/query"
)

func TestBlockedQueryMatchesBlockersClosedGuard(t *testing.T) {
	cfg := &datamodel.Config{Workflows: map[string]datamodel.Workflow{
		datamodel.TypeTicket: {States: []datamodel.State{
			{Key: "TODO", Category: datamodel.CategoryTodo},
			{Key: "IN_PROGRESS", Category: datamodel.CategoryDoing},
			{Key: "DONE", Category: datamodel.CategoryDone},
		}},
	}}

	doingBlocker := &datamodel.Item{ID: "B-doing", Number: "KIRA-2", Type: datamodel.TypeTicket, State: "IN_PROGRESS"}
	doneBlocker := &datamodel.Item{ID: "B-done", Number: "KIRA-3", Type: datamodel.TypeTicket, State: "DONE"}
	orphanBlocker := &datamodel.Item{ID: "B-orphan", Number: "KIRA-4", Type: "saga", State: "TODO"}
	items := []*datamodel.Item{doingBlocker, doneBlocker, orphanBlocker}

	cases := []struct {
		name        string
		blockedBy   string
		wantBlocked bool
		wantHard    bool
		wantWarn    string
	}{
		{"missing blocker", "B-missing", false, false, "resolves to no item"},
		{"orphan-type blocker", orphanBlocker.ID, false, false, "no known state category"},
		{"doing blocker (open)", doingBlocker.ID, true, true, ""},
		{"done blocker (closed)", doneBlocker.ID, false, false, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			subject := &datamodel.Item{ID: "SUBJECT", Type: datamodel.TypeTicket, State: "TODO", BlockedBy: []string{c.blockedBy}}
			all := append(append([]*datamodel.Item{}, items...), subject)

			compiled, err := query.Compile("blocked", query.Options{Items: all})
			if err != nil {
				t.Fatalf("Compile: %v", err)
			}
			if got := compiled.Pred(subject, cfg); got != c.wantBlocked {
				t.Errorf("query blocked = %v, want %v", got, c.wantBlocked)
			}

			hard, warns := blockersClosedGuard(cfg, subject, all, "TODO", "IN_PROGRESS", false)
			if gotHard := len(hard) > 0; gotHard != c.wantHard {
				t.Errorf("guard hard-blocked = %v, want %v (hard=%v)", gotHard, c.wantHard, hard)
			}
			if c.wantWarn != "" {
				found := false
				for _, w := range warns {
					if strings.Contains(w.Error(), c.wantWarn) {
						found = true
					}
				}
				if !found {
					t.Errorf("warns = %v, want one containing %q", warns, c.wantWarn)
				}
			}
		})
	}
}
