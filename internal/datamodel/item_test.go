package datamodel

import (
	"slices"
	"testing"
)

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

func TestMutableFieldsOrder(t *testing.T) {
	want := []string{
		KeySubtype, KeyTitle, KeyResolution, KeyPriority, KeyRank, KeyOwner,
		KeyReporter, KeyLabels, KeyEpic, KeySprint, KeyDue, KeyEstimate,
	}
	if !slices.Equal(MutableFields, want) {
		t.Fatalf("MutableFields = %v, want %v", MutableFields, want)
	}
}
