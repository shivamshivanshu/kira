package core_test

import (
	"strings"
	"testing"
)

func TestBoardCreateRejectsReservedSubcommandKey(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)

	_, err := s.BoardCreate(cfg, "UNARCHIVE", "Unarchive", "")
	if err == nil {
		t.Fatal("creating a board keyed on the unarchive subcommand must be rejected")
	}
	if !strings.Contains(err.Error(), "reserved subcommand name") {
		t.Fatalf("rejection message = %q, want 'reserved subcommand name'", err.Error())
	}
}
