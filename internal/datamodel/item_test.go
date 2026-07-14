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
