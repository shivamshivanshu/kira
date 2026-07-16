package core

import "testing"

// A stale cfg snapshot must not let BoardArchive approve archiving what is,
// on the freshly-locked config, actually the last active board.
func TestBoardArchiveGuardsAgainstStaleConfig(t *testing.T) {
	s, cfg, _ := stagedFixture(t)
	if _, err := s.BoardCreate(cfg, "SECOND", "Second", ""); err != nil {
		t.Fatalf("BoardCreate: %v", err)
	}

	stale, err := s.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if len(stale.ActiveBoards()) != 2 {
		t.Fatalf("active boards = %d, want 2", len(stale.ActiveBoards()))
	}

	if _, err := s.BoardArchive(cfg, stale.Project.Key); err != nil {
		t.Fatalf("BoardArchive(first): %v", err)
	}

	if _, err := s.BoardArchive(stale, "SECOND"); err == nil {
		t.Fatal("BoardArchive must refuse to archive the last active board, even against a stale cfg snapshot")
	}
}
