package core

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
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
