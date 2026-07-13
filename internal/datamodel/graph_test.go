package datamodel

import (
	"reflect"
	"testing"
)

func blockedByEdges(it *Item) []string { return it.BlockedBy }

func itemsByID(items ...*Item) map[string]*Item {
	byID := make(map[string]*Item, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}
	return byID
}

func TestFindCycles(t *testing.T) {
	tests := []struct {
		name  string
		items []*Item
		want  [][]string
	}{
		{
			name:  "acyclic chain",
			items: []*Item{{ID: "A", BlockedBy: []string{"B"}}, {ID: "B", BlockedBy: []string{"C"}}, {ID: "C"}},
			want:  nil,
		},
		{
			name:  "two node cycle",
			items: []*Item{{ID: "A", BlockedBy: []string{"B"}}, {ID: "B", BlockedBy: []string{"A"}}},
			want:  [][]string{{"A", "B"}},
		},
		{
			name:  "self reference",
			items: []*Item{{ID: "A", BlockedBy: []string{"A"}}},
			want:  [][]string{{"A"}},
		},
		{
			name:  "three node cycle",
			items: []*Item{{ID: "A", BlockedBy: []string{"B"}}, {ID: "B", BlockedBy: []string{"C"}}, {ID: "C", BlockedBy: []string{"A"}}},
			want:  [][]string{{"A", "B", "C"}},
		},
		{
			name:  "edge to missing node is ignored",
			items: []*Item{{ID: "A", BlockedBy: []string{"Z"}}},
			want:  nil,
		},
		{
			name:  "cycle reported once despite extra feeder",
			items: []*Item{{ID: "A", BlockedBy: []string{"B"}}, {ID: "B", BlockedBy: []string{"A"}}, {ID: "D", BlockedBy: []string{"A"}}},
			want:  [][]string{{"A", "B"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindCycles(itemsByID(tt.items...), blockedByEdges)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("FindCycles = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCycleRelations(t *testing.T) {
	rels := CycleRelations()
	if len(rels) != 2 {
		t.Fatalf("CycleRelations len = %d, want 2", len(rels))
	}
	it := &Item{
		BlockedBy: []string{"A"},
		Links:     map[string][]string{string(LinkDuplicateOf): {"B"}},
	}
	if rels[0].Field != KeyBlockedBy {
		t.Errorf("rels[0].Field = %q, want %q", rels[0].Field, KeyBlockedBy)
	}
	if got := rels[0].Edges(it); !reflect.DeepEqual(got, []string{"A"}) {
		t.Errorf("blocked_by edges = %v, want [A]", got)
	}
	if rels[1].Field != string(LinkDuplicateOf) {
		t.Errorf("rels[1].Field = %q, want %q", rels[1].Field, LinkDuplicateOf)
	}
	if got := rels[1].Edges(it); !reflect.DeepEqual(got, []string{"B"}) {
		t.Errorf("duplicate_of edges = %v, want [B]", got)
	}
}
