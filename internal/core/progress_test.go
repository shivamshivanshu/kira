package core

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func progressCfg() *datamodel.Config {
	return &datamodel.Config{ResolutionsDropped: []string{datamodel.ResolutionDropped}, Workflows: map[string]datamodel.Workflow{
		datamodel.TypeTicket: {States: []datamodel.State{
			{Key: "TODO", Category: datamodel.CategoryTodo},
			{Key: "DONE", Category: datamodel.CategoryDone},
		}},
		datamodel.TypeEpic: {States: []datamodel.State{
			{Key: "OPEN", Category: datamodel.CategoryDoing},
		}},
	}}
}

// epicContainers is what a default repo's schemas declare: epic is the only
// type carrying the container capability.
func epicContainers() map[string]bool { return map[string]bool{datamodel.TypeEpic: true} }

func ticket(id, state string, resolution *string) *datamodel.Item {
	return &datamodel.Item{ID: id, Type: datamodel.TypeTicket, State: state, Resolution: resolution}
}

func TestEpicProgressDroppedCountsAsDone(t *testing.T) {
	dropped := datamodel.ResolutionDropped
	children := map[string][]*datamodel.Item{
		"E1": {
			ticket("T1", "TODO", nil),
			ticket("T2", "DONE", nil),
			ticket("T3", "DONE", &dropped),
			{ID: "E2", Type: datamodel.TypeEpic, State: "OPEN"},
		},
		"E2": {ticket("T4", "DONE", nil)},
	}
	p, err := epicProgress(progressCfg(), children, "E1", epicContainers())
	if err != nil {
		t.Fatalf("epicProgress: %v", err)
	}
	if p.Total != 4 {
		t.Errorf("total = %d, want 4", p.Total)
	}
	if p.Done != 3 {
		t.Errorf("done = %d, want 3 (dropped counts as done, sub-epic recursed)", p.Done)
	}
}

func TestEpicProgressDoesNotDescendNonEpicParent(t *testing.T) {
	children := map[string][]*datamodel.Item{
		"E1": {ticket("T1", "DONE", nil)},
		"T1": {ticket("T2", "DONE", nil)},
	}
	p, err := epicProgress(progressCfg(), children, "E1", epicContainers())
	if err != nil {
		t.Fatalf("epicProgress: %v", err)
	}
	if p.Total != 1 || p.Done != 1 {
		t.Errorf("got %d/%d, want 1/1 (non-epic parent T1 must not be descended)", p.Done, p.Total)
	}
}

func TestEpicProgressCycleSafe(t *testing.T) {
	children := map[string][]*datamodel.Item{
		"E1": {{ID: "E2", Type: datamodel.TypeEpic, State: "OPEN"}, ticket("T1", "DONE", nil)},
		"E2": {{ID: "E1", Type: datamodel.TypeEpic, State: "OPEN"}},
	}
	p, err := epicProgress(progressCfg(), children, "E1", epicContainers())
	if err == nil {
		t.Fatal("expected a cycle error for E1 <-> E2")
	}
	if p.Total != 1 || p.Done != 1 {
		t.Errorf("got %d/%d, want 1/1 without infinite recursion", p.Done, p.Total)
	}
}

// The capability, not the type name, is what makes a type roll up its
// children -- so a type that is not "epic" must behave identically, and "epic"
// without the capability must stop.
func TestProgressFollowsTheContainerCapabilityNotTheTypeName(t *testing.T) {
	const feature = "feature"
	children := map[string][]*datamodel.Item{
		"F1": {ticket("T1", "DONE", nil), {ID: "F2", Type: feature, State: "OPEN"}},
		"F2": {ticket("T2", "TODO", nil)},
	}
	cfg := progressCfg()
	cfg.Workflows[feature] = datamodel.Workflow{States: []datamodel.State{{Key: "OPEN", Category: datamodel.CategoryDoing}}}

	p, err := epicProgress(cfg, children, "F1", map[string]bool{feature: true})
	if err != nil {
		t.Fatalf("epicProgress: %v", err)
	}
	if p.Total != 2 || p.Done != 1 {
		t.Errorf("got %d/%d, want 1/2 -- a declared container must recurse whatever it is called", p.Done, p.Total)
	}
}

func TestProgressStopsAtATypeThatNoLongerDeclaresTheCapability(t *testing.T) {
	children := map[string][]*datamodel.Item{
		"E1": {ticket("T1", "DONE", nil), {ID: "E2", Type: datamodel.TypeEpic, State: "OPEN"}},
		"E2": {ticket("T2", "DONE", nil)},
	}

	p, err := epicProgress(progressCfg(), children, "E1", map[string]bool{})
	if err != nil {
		t.Fatalf("epicProgress: %v", err)
	}
	if p.Total != 2 {
		t.Errorf("total = %d, want 2 -- with no container declared, E2 counts as a plain child and is not descended", p.Total)
	}
}
