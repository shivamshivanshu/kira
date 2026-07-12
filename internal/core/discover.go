package core

import (
	"os/exec"

	"github.com/shivamshivanshu/kira/internal/id"
)

// Candidate is one selectable row in the discover picker: the display number
// and title a human scans, plus the ULID the selection resolves to.
type Candidate struct {
	ID     string
	Number string
	Title  string
}

// HaveFzf reports whether fzf is on PATH, detected at call time per the
// external-tool policy (docs/design/01-architecture.md §7).
func HaveFzf() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
}

// Candidates returns the discover picker's source list — every item as a
// {number, title} row — sorted by display number, the same data list --json
// exposes (docs/design/04-cli.md discover). The picker UI (fzf/bubbles) lives
// in the cli layer; this is just the data feed.
func (s *Store) Candidates() ([]Candidate, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	out := make([]Candidate, len(items))
	for i, it := range items {
		out[i] = Candidate{ID: it.ID, Number: it.Number, Title: it.Title}
	}
	sortByKey(out, func(c Candidate) id.SortKey { return id.NewSortKey(c.Number, c.ID) })
	return out, nil
}
