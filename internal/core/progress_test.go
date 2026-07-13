package core

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func progressCfg() *datamodel.Config {
	return &datamodel.Config{Workflows: map[string]datamodel.Workflow{
		datamodel.TypeTicket: {States: []datamodel.State{
			{Key: "TODO", Category: datamodel.CategoryTodo},
			{Key: "DONE", Category: datamodel.CategoryDone},
		}},
		datamodel.TypeEpic: {States: []datamodel.State{
			{Key: "OPEN", Category: datamodel.CategoryDoing},
		}},
	}}
}

func ticket(id, state string, resolution *string) *datamodel.Item {
	return &datamodel.Item{ID: id, Type: datamodel.TypeTicket, State: state, Resolution: resolution}
}

func TestAccumulateProgressDroppedExcludedFromNumerator(t *testing.T) {
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
	var p datamodel.EpicProgress
	accumulateProgress(progressCfg(), children, "E1", map[string]bool{}, &p)
	if p.Total != 4 {
		t.Errorf("total = %d, want 4 (dropped counts toward denominator)", p.Total)
	}
	if p.Done != 2 {
		t.Errorf("done = %d, want 2 (dropped excluded from numerator, sub-epic recursed)", p.Done)
	}
}

func TestAccumulateProgressCycleSafe(t *testing.T) {
	children := map[string][]*datamodel.Item{
		"E1": {{ID: "E2", Type: datamodel.TypeEpic, State: "OPEN"}, ticket("T1", "DONE", nil)},
		"E2": {{ID: "E1", Type: datamodel.TypeEpic, State: "OPEN"}},
	}
	var p datamodel.EpicProgress
	accumulateProgress(progressCfg(), children, "E1", map[string]bool{}, &p)
	if p.Total != 1 || p.Done != 1 {
		t.Errorf("got %d/%d, want 1/1 without infinite recursion", p.Done, p.Total)
	}
}
