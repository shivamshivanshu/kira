package core

import (
	"github.com/shivamshivanshu/kira/internal/id"
)

type Candidate struct {
	ID     string
	Number string
	Title  string
}

func (s *Store) Candidates() ([]Candidate, error) {
	items, _, err := s.LoadAll()
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
