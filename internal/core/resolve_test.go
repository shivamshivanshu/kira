package core

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

type scriptedPrompter struct {
	answers []string
	pos     int
}

func (p *scriptedPrompter) Interactive() bool { return true }

func (p *scriptedPrompter) Confirm(string) bool { return false }

func (p *scriptedPrompter) ReadLine(_, def string) string {
	if p.pos >= len(p.answers) {
		return def
	}
	a := p.answers[p.pos]
	p.pos++
	return a
}

func TestResolveRefusesWhenLockAlreadyHeld(t *testing.T) {
	s, _ := seededSyncRepo(t)
	release, err := s.fs().Lock()
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	defer release()

	_, err = s.Resolve(nil, false)
	var ce *errx.Error
	if !errors.As(err, &ce) || ce.Code != errx.ExitConflict {
		t.Fatalf("err = %v, want errx.Conflict", err)
	}
}

func TestResolveSkipsFilenameIDMismatchInsteadOfMisreportingResolved(t *testing.T) {
	s, repo := seededSyncRepo(t)

	mismatched := eventTicket()
	mismatched.ID = "01HZZ0TEST0000000000000099"
	mismatched.Number = "KIRA-9"
	wrongPath := filepath.Join(s.root, ".kira", "tickets", "01HZZ0TEST0000000000000000.md")
	writeAndCommit := func(state, date string) {
		mismatched.State = state
		mismatched.Updated = date + "T10:00:00Z"
		if err := os.WriteFile(wrongPath, []byte(codec.Serialize(mismatched)), 0o644); err != nil {
			t.Fatal(err)
		}
		gitRun(t, s, date+"T10:00:00Z", "add", "-A")
		gitRun(t, s, date+"T10:00:00Z", "commit", "-m", "state "+state)
	}
	writeAndCommit("TODO", "2026-02-01")

	mismatched.State = "IN_PROGRESS"
	mismatched.Updated = "2026-02-02T10:00:00Z"
	if err := os.WriteFile(wrongPath, []byte(codec.Serialize(mismatched)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := repo.Stash(); err != nil {
		t.Fatalf("stash: %v", err)
	}
	writeAndCommit("REVIEW", "2026-02-03")

	if err := repo.StashPop(); err == nil {
		t.Fatal("expected a stash pop conflict on the mismatched file")
	}

	wantRel := ".kira/tickets/01HZZ0TEST0000000000000000.md"
	res, err := s.Resolve(nil, false)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(res.Resolved) != 0 {
		t.Fatalf("Resolved = %+v, want none reported resolved for a filename/ID mismatch", res.Resolved)
	}
	if !slices.Contains(res.Skipped, wantRel) {
		t.Fatalf("Skipped = %v, want it to include %s", res.Skipped, wantRel)
	}
	unmerged, err := repo.UnmergedPaths()
	if err != nil {
		t.Fatalf("unmerged: %v", err)
	}
	if !slices.Contains(unmerged, wantRel) {
		t.Errorf("unmerged = %v, want the original conflict still standing at %s", unmerged, wantRel)
	}
}

func TestPickFieldsHonoursSideChoices(t *testing.T) {
	s := &Store{prompter: &scriptedPrompter{answers: []string{"o", "t", ""}}}

	target := eventTicket()
	ours := eventTicket()
	ours.Title = "ours-title"
	ours.Priority = ptr.To("P0")
	ours.Owner = ptr.To("ours-owner")
	theirs := eventTicket()
	theirs.Title = "theirs-title"
	theirs.Priority = ptr.To("P1")
	theirs.Owner = ptr.To("theirs-owner")

	s.pickFields(target, ours, theirs, []string{datamodel.KeyTitle, datamodel.KeyPriority, datamodel.KeyOwner})

	if target.Title != "ours-title" {
		t.Errorf("title = %q, want ours pick", target.Title)
	}
	if target.Priority == nil || *target.Priority != "P1" {
		t.Errorf("priority = %v, want theirs pick", target.Priority)
	}
	if target.Owner != nil {
		t.Errorf("owner = %v, want untouched (nil) on empty/auto answer", target.Owner)
	}
}
