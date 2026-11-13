package core

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func ticket(id, state string, resolution *string) *datamodel.Item {
	return &datamodel.Item{ID: id, Type: datamodel.TypeTicket, State: state, Resolution: resolution}
}

func under(parent string, it *datamodel.Item) *datamodel.Item {
	it.Parent = &parent
	return it
}

func container(id, typ string) *datamodel.Item {
	return &datamodel.Item{ID: id, Type: typ, State: "OPEN"}
}

func TestProgressCountsDoneStatesAcrossNestedContainers(t *testing.T) {
	dropped := datamodel.ResolutionDropped
	items := []*datamodel.Item{
		container("E1", datamodel.TypeEpic),
		under("E1", ticket("T1", "TODO", nil)),
		under("E1", ticket("T2", "DONE", nil)),
		under("E1", ticket("T3", "DONE", &dropped)),
		under("E1", container("E2", datamodel.TypeEpic)),
		under("E2", ticket("T4", "DONE", nil)),
	}

	got, err := allContainerProgress(items)
	if err != nil {
		t.Fatalf("allContainerProgress: %v", err)
	}

	if p := got["E1"]; p.Total != 4 || p.Done != 3 {
		t.Errorf("E1 = %d/%d, want 3/4 -- the nested container's own children roll up", p.Done, p.Total)
	}
	if p := got["E2"]; p.Total != 1 || p.Done != 1 {
		t.Errorf("E2 = %d/%d, want 1/1", p.Done, p.Total)
	}
}

func TestProgressDoesNotDescendANonContainerChild(t *testing.T) {
	items := []*datamodel.Item{
		container("E1", datamodel.TypeEpic),
		under("E1", ticket("T1", "DONE", nil)),
		under("T1", ticket("T2", "DONE", nil)),
	}

	got, err := allContainerProgress(items)
	if err != nil {
		t.Fatalf("allContainerProgress: %v", err)
	}

	if p := got["E1"]; p.Total != 1 || p.Done != 1 {
		t.Errorf("E1 = %d/%d, want 1/1 -- T1 is not a container, so T2 beneath it is not counted", p.Done, p.Total)
	}
}

func TestProgressReportsAParentCycleInsteadOfRecursingForever(t *testing.T) {
	items := []*datamodel.Item{
		under("E2", container("E1", datamodel.TypeEpic)),
		under("E1", container("E2", datamodel.TypeEpic)),
		under("E1", ticket("T1", "DONE", nil)),
	}

	_, err := allContainerProgress(items)

	if err == nil {
		t.Fatal("expected a cycle error for E1 <-> E2")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error = %q, want it to name the cycle", err)
	}
}
