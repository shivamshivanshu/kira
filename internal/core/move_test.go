package core

import (
	"strings"
	"testing"
)

func TestMoveSubjectHandlesPercentInPrefix(t *testing.T) {
	s, cfg, repo := stagedFixture(t)
	cfg.Commit.SubjectPrefix = "100% "

	res, err := s.Create(cfg, CreateOpts{Type: "ticket", Title: "T", NoEdit: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.Move(cfg, res.Number, "IN_PROGRESS", MoveOpts{}); err != nil {
		t.Fatalf("move: %v", err)
	}

	msg, err := repo.Output("log", "-1", "--format=%B")
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if strings.Contains(msg, "MISSING") {
		t.Fatalf("commit subject corrupted by a %%-containing prefix: %q", msg)
	}
	want := "100% " + res.Number + " state TODO -> IN_PROGRESS"
	if !strings.Contains(msg, want) {
		t.Errorf("commit message = %q, want it to contain %q", msg, want)
	}
}
