package core

import (
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/timex"
)

func (s *Store) Log(cfg *datamodel.Config, ref string) (*datamodel.LogResult, error) {
	ld, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		return nil, err
	}
	it, err := findItem(ld.items, ld.resolver, ref)
	if err != nil {
		return nil, err
	}

	events, links, err := s.logEntries(it.ID)
	if err != nil {
		return nil, err
	}
	return &datamodel.LogResult{ID: it.ID, Number: it.Number, Entries: interleave(events, links)}, nil
}

func (s *Store) logEntries(ulid string) ([]datamodel.Event, []index.CommitLink, error) {
	return index.LogEntries(s.fs(), ulid, s.fileHead(ulid), func() ([]datamodel.Event, error) {
		return s.deriveEvents(ulid)
	})
}

func interleave(events []datamodel.Event, links []index.CommitLink) []datamodel.LogEntry {
	entries := make([]datamodel.LogEntry, 0, len(events)+len(links))
	for _, e := range events {
		entries = append(entries, datamodel.LogEntry{
			Kind: datamodel.LogKindEvent, Ts: e.Ts, Field: e.Field, Old: e.Old, New: e.New, SHA: e.CommitSHA,
		})
	}
	for _, l := range links {
		entries = append(entries, datamodel.LogEntry{
			Kind: datamodel.LogKindCommit, Ts: l.Ts, SHA: l.SHA, Subject: l.Subject, Author: l.Author,
		})
	}
	slices.SortStableFunc(entries, func(a, b datamodel.LogEntry) int {
		c, aOK, bOK := timex.CompareRFC3339(a.Ts, b.Ts)
		if aOK && bOK {
			return -c
		}
		return strings.Compare(b.Ts, a.Ts)
	})
	return entries
}
