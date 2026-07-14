package core

import "testing"

func TestWithPrompterReplacesOnCopy(t *testing.T) {
	s := &Store{prompter: &scriptedPrompter{}}
	silenced := s.WithPrompter(SilentPrompter())
	if silenced.prompter.Interactive() {
		t.Fatal("WithPrompter(SilentPrompter()) should yield a non-interactive store")
	}
	if !s.prompter.Interactive() {
		t.Fatal("WithPrompter must not mutate the receiver")
	}
	if s.WithPrompter(nil).prompter.Interactive() {
		t.Fatal("WithPrompter(nil) should fall back to the silent prompter")
	}
}
