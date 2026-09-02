package datamodel

import (
	"slices"
	"testing"
)

func TestIndexByID(t *testing.T) {
	a := &Item{ID: "a"}
	b := &Item{ID: "b"}
	blank := &Item{}

	got := IndexByID([]*Item{a, b, blank})

	want := map[string]*Item{"a": a, "b": b}
	if len(got) != len(want) || got["a"] != a || got["b"] != b {
		t.Fatalf("IndexByID(...) = %v, want %v", got, want)
	}
}

func TestFrontmatterKeysOrder(t *testing.T) {
	want := []string{
		KeyID, KeyNumber, KeyAliases, KeyType, KeySubtype, KeyTitle, KeyState,
		KeyResolution, KeyPriority, KeyRank, KeyOwner, KeyReporter, KeyLabels,
		KeyParent, KeyBlockedBy, KeyLinks, KeySprint, KeyDue, KeyEstimate,
		KeyCreated, KeyUpdated,
	}
	if !slices.Equal(FrontmatterKeys, want) {
		t.Fatalf("FrontmatterKeys = %v, want %v", FrontmatterKeys, want)
	}
}

func TestChangedExtraKeys(t *testing.T) {
	tests := []struct {
		name               string
		aScalars, bScalars map[string]Rendered
		aLists, bLists     map[string][]Rendered
		want               []string
	}{
		{
			name:     "equal",
			aScalars: map[string]Rendered{"team": "infra"}, bScalars: map[string]Rendered{"team": "infra"},
			aLists: map[string][]Rendered{"watchers": {"a"}}, bLists: map[string][]Rendered{"watchers": {"a"}},
		},
		{
			name:     "added",
			bScalars: map[string]Rendered{"team": "infra"},
			want:     []string{"team"},
		},
		{
			name:     "removed",
			aScalars: map[string]Rendered{"team": "infra"},
			want:     []string{"team"},
		},
		{
			name:     "modified scalar",
			aScalars: map[string]Rendered{"team": "infra"}, bScalars: map[string]Rendered{"team": "core"},
			want: []string{"team"},
		},
		{
			name:   "modified list",
			aLists: map[string][]Rendered{"watchers": {"a"}}, bLists: map[string][]Rendered{"watchers": {"a", "b"}},
			want: []string{"watchers"},
		},
		{
			name:     "independent keys sorted",
			aScalars: map[string]Rendered{"k2": "y"}, bScalars: map[string]Rendered{"k1": "x"},
			want: []string{"k1", "k2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Item{ExtraScalars: tt.aScalars, ExtraLists: tt.aLists}
			b := &Item{ExtraScalars: tt.bScalars, ExtraLists: tt.bLists}
			if got := ChangedExtraKeys(a, b); !slices.Equal(got, tt.want) {
				t.Fatalf("ChangedExtraKeys = %v, want %v", got, tt.want)
			}
		})
	}
}
