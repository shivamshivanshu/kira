package config

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestParseTicketStates(t *testing.T) {
	cfg, err := Parse([]byte(`ticket_states:
  bug:
    - {name: Reported, category: todo}
    - {name: Ready for review, category: doing}
    - {name: Verified, category: done}
`))
	if err != nil {
		t.Fatal(err)
	}
	bug, _ := cfg.ItemWorkflow(datamodel.TypeTicket, "bug")
	if !bug.Custom || bug.Initial != "Reported" || bug.CloseTarget() != "Verified" {
		t.Fatalf("bug workflow = %+v", bug)
	}
	state, ok := bug.StateByKey("Ready for review")
	if !ok || state.Category != datamodel.CategoryDoing {
		t.Fatalf("custom state = %+v, found = %t", state, ok)
	}
	task, _ := cfg.ItemWorkflow(datamodel.TypeTicket, "task")
	if task.Custom || task.Initial != "TODO" || task.CloseTarget() != "DONE" {
		t.Fatalf("default workflow = %+v", task)
	}
	epic, _ := cfg.ItemWorkflow(datamodel.TypeEpic, "bug")
	if epic.Custom || epic.Initial != "PLANNED" || epic.CloseTarget() != "DONE" {
		t.Fatalf("epic workflow = %+v", epic)
	}
}

func TestRejectInvalidTicketStates(t *testing.T) {
	for _, test := range []struct {
		name, states, errorText string
	}{
		{"unknown subtype", "incident: [{name: Closed, category: done}]", "unknown ticket subtype"},
		{"empty workflow", "task: []", "at least one state"},
		{"invalid name", "task: [{name: ' Closed', category: done}]", "single-line name"},
		{"invalid category", "task: [{name: Closed, category: complete}]", ".category"},
		{"duplicate name", "task: [{name: X, category: todo}, {name: X, category: done}]", "duplicate state"},
		{"last state open", "task: [{name: X, category: doing}]", "last state must have category done"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte("ticket_states:\n  " + test.states + "\n"))
			if err == nil || !strings.Contains(err.Error(), test.errorText) {
				t.Fatalf("Parse error = %v, want %q", err, test.errorText)
			}
		})
	}
}
