package core

import (
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestBlockersRanksDistinctDescendantTickets(t *testing.T) {
	epic := &datamodel.Item{ID: "epic", Number: "KIRA-1", Type: datamodel.TypeEpic, State: "ACTIVE", BlockedBy: []string{"direct"}}
	nested := &datamodel.Item{ID: "nested", Number: "KIRA-2", Type: datamodel.TypeEpic, State: "ACTIVE", Parent: &epic.ID}
	first := &datamodel.Item{ID: "first", Number: "KIRA-3", Type: datamodel.TypeTicket, State: "TODO", Parent: &epic.ID, BlockedBy: []string{"left", "right", "higher", "lower"}}
	second := &datamodel.Item{ID: "second", Number: "KIRA-4", Type: datamodel.TypeTicket, State: "TODO", Parent: &nested.ID, BlockedBy: []string{"shared"}}
	closed := &datamodel.Item{ID: "closed", Number: "KIRA-5", Type: datamodel.TypeTicket, State: "DONE", Parent: &epic.ID, BlockedBy: []string{"shared"}}
	left := &datamodel.Item{ID: "left", Number: "KIRA-6", Type: datamodel.TypeTicket, State: "TODO", BlockedBy: []string{"shared"}}
	right := &datamodel.Item{ID: "right", Number: "KIRA-7", Type: datamodel.TypeTicket, State: "TODO", BlockedBy: []string{"shared"}}
	shared := &datamodel.Item{ID: "shared", Number: "CORE-30", Type: datamodel.TypeTicket, State: "TODO"}
	lower := &datamodel.Item{ID: "lower", Number: "KIRA-8", Type: datamodel.TypeTicket, State: "TODO"}
	higher := &datamodel.Item{ID: "higher", Number: "KIRA-10", Type: datamodel.TypeTicket, State: "TODO"}
	direct := &datamodel.Item{ID: "direct", Number: "CORE-40", Type: datamodel.TypeTicket, State: "TODO"}

	result, err := blockerReport(nil, epic, []*datamodel.Item{epic, nested, second, first, closed, left, right, shared, lower, higher, direct})
	if err != nil {
		t.Fatal(err)
	}
	wantIDs := []string{"shared", "lower", "higher"}
	wantAffected := []int{2, 1, 1}
	if len(result.Actionable) != len(wantIDs) {
		t.Fatalf("actionable blockers: %+v", result.Actionable)
	}
	for idx, blocker := range result.Actionable {
		if blocker.Item.ID != wantIDs[idx] || blocker.Affected != wantAffected[idx] {
			t.Errorf("blocker %d = %+v, want %s affecting %d", idx, blocker, wantIDs[idx], wantAffected[idx])
		}
	}
	if path := result.Actionable[0].Path; !slices.Equal(path, []string{"KIRA-3", "KIRA-6", "CORE-30"}) {
		t.Errorf("shared blocker path = %v", path)
	}
	if len(result.Direct) != 1 || result.Direct[0].Item.ID != direct.ID || result.Direct[0].Affected != 0 {
		t.Errorf("direct epic blockers: %+v", result.Direct)
	}
}

func TestBlockersKeepsActionablePathAlongsideIncompleteChains(t *testing.T) {
	cfg := &datamodel.Config{TicketStates: map[string][]datamodel.State{
		"bug": {
			{Key: "Fixing", Category: datamodel.CategoryDoing},
			{Key: "Verified", Category: datamodel.CategoryDone},
		},
	}}
	bug := "bug"
	root := &datamodel.Item{ID: "root", Number: "KIRA-1", Type: datamodel.TypeTicket, Subtype: &bug, State: "Fixing", BlockedBy: []string{"cycle", "unknown", "missing", "closed", "dropped", "leaf"}}
	cycle := &datamodel.Item{ID: "cycle", Number: "KIRA-2", Type: datamodel.TypeTicket, State: "TODO", BlockedBy: []string{"root"}}
	unknown := &datamodel.Item{ID: "unknown", Number: "KIRA-3", Type: datamodel.TypeTicket, Subtype: &bug, State: "UNCONFIGURED"}
	closed := &datamodel.Item{ID: "closed", Number: "KIRA-4", Type: datamodel.TypeTicket, Subtype: &bug, State: "Verified", BlockedBy: []string{"hidden"}}
	dropped := &datamodel.Item{ID: "dropped", Number: "KIRA-5", Type: datamodel.TypeTicket, State: "WONT_DO", BlockedBy: []string{"hidden"}}
	hidden := &datamodel.Item{ID: "hidden", Number: "KIRA-6", Type: datamodel.TypeTicket, State: "TODO"}
	leaf := &datamodel.Item{ID: "leaf", Number: "KIRA-7", Type: datamodel.TypeTicket, Subtype: &bug, State: "Fixing"}
	byID := datamodel.IndexByID([]*datamodel.Item{root, cycle, unknown, closed, dropped, hidden, leaf})

	actionable, incomplete := searchBlockers(cfg, root, byID)
	if len(actionable) != 1 || actionable[0].Item.ID != leaf.ID || !slices.Equal(actionable[0].Path, []string{"KIRA-1", "KIRA-7"}) {
		t.Errorf("actionable blockers: %+v", actionable)
	}
	want := []datamodel.BlockerIssue{
		{Path: []string{"KIRA-1", "KIRA-3"}, Reason: "has no known state category"},
		{Path: []string{"KIRA-1", "missing"}, Reason: "resolves to no item"},
		{Path: []string{"KIRA-1", "KIRA-2", "KIRA-1"}, Reason: "blocker cycle"},
	}
	if !slices.EqualFunc(incomplete, want, func(got, want datamodel.BlockerIssue) bool {
		return got.Reason == want.Reason && slices.Equal(got.Path, want.Path)
	}) {
		t.Errorf("incomplete chains = %+v, want %+v", incomplete, want)
	}

	root.State = "Verified"
	actionable, incomplete = searchBlockers(cfg, root, byID)
	if len(actionable)+len(incomplete) != 0 {
		t.Errorf("closed ticket has blockers: actionable %+v, incomplete %+v", actionable, incomplete)
	}
}
