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
		KeyEpic, KeyBlockedBy, KeyLinks, KeySprint, KeyDue, KeyEstimate,
		KeyCreated, KeyUpdated,
	}
	if !slices.Equal(FrontmatterKeys, want) {
		t.Fatalf("FrontmatterKeys = %v, want %v", FrontmatterKeys, want)
	}
}
