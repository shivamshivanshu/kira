package core

import (
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/timex"
)

var scalarFields = scalarFieldSet()

func scalarFieldSet() map[string]bool {
	list := map[string]bool{
		datamodel.KeyAliases: true, datamodel.KeyLabels: true,
		datamodel.KeyBlockedBy: true, datamodel.KeyLinks: true,
		datamodel.KeyCreated: true, datamodel.KeyUpdated: true,
	}
	scalar := map[string]bool{}
	for _, k := range datamodel.FrontmatterKeys {
		if !list[k] {
			scalar[k] = true
		}
	}
	return scalar
}

func (s *Store) Log(cfg *datamodel.Config, ref string) (*datamodel.LogResult, error) {
	ld, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		return nil, err
	}
	items, resolver := ld.items, ld.resolver
	ulid, err := resolveID(resolver, ref)
	if err != nil {
		return nil, err
	}
	it := findByULID(items, ulid)
	if it == nil {
		return nil, errx.User("resolved %s to %s, which has no file", ref, ulid)
	}

	events, links, err := index.LogEntries(s.fs(), ulid, s.fileHead(ulid), func() ([]datamodel.Event, error) {
		return s.deriveEvents(ulid)
	})
	if err != nil {
		return nil, err
	}
	return &datamodel.LogResult{ID: ulid, Number: it.Number, Entries: interleave(events, links)}, nil
}

func (s *Store) deriveEvents(ulid string) ([]datamodel.Event, error) {
	out, err := s.repo().FileLog(s.fs().RelToRoot(s.itemPath(ulid)))
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits") {
			return nil, nil
		}
		return nil, errx.User("%s", err)
	}
	var events []datamodel.Event
	walkPatch(out, func(_ string, evs []datamodel.Event) {
		events = append(events, evs...)
	})
	return events, nil
}

func fmEvents(created bool, minus, plus map[string]string, ts, sha string) []datamodel.Event {
	if created {
		return nil
	}
	var events []datamodel.Event
	for _, field := range datamodel.FrontmatterKeys {
		mv, hadMinus := minus[field]
		pv, hadPlus := plus[field]
		if (hadMinus || hadPlus) && mv != pv {
			events = append(events, datamodel.Event{Ts: ts, Field: field, Old: mv, New: pv, CommitSHA: sha})
		}
	}
	return events
}

func frontmatterField(line string) (key, value string, ok bool) {
	colon := strings.IndexByte(line, ':')
	if colon <= 0 {
		return "", "", false
	}
	key = line[:colon]
	if !scalarFields[key] {
		return "", "", false
	}
	return key, unquoteScalar(strings.TrimSpace(line[colon+1:])), true
}

func unquoteScalar(v string) string {
	return strings.Trim(strings.TrimSpace(v), `"'`)
}

func interleave(events []datamodel.Event, links []index.CommitLink) []datamodel.LogEntry {
	entries := make([]datamodel.LogEntry, 0, len(events)+len(links))
	for _, e := range events {
		entries = append(entries, datamodel.LogEntry{
			Kind: "event", Ts: e.Ts, Field: e.Field, Old: e.Old, New: e.New, SHA: e.CommitSHA,
		})
	}
	for _, l := range links {
		entries = append(entries, datamodel.LogEntry{
			Kind: "commit", Ts: l.Ts, SHA: l.SHA, Subject: l.Subject, Author: l.Author,
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
