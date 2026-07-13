package core

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
)

const historyTailLimit = 10

func (s *Store) Show(cfg *datamodel.Config, ref, at string) (*datamodel.ShowResult, error) {
	ld, err := s.read(cfg, loadOpts{at: at, useIndex: true})
	if err != nil {
		return nil, err
	}
	ulid, err := resolveID(ld.resolver, ref)
	if err != nil {
		return nil, err
	}
	if at != "" {
		it := findByULID(ld.items, ulid)
		if it == nil {
			return nil, errx.User("%s resolved to %s, which is absent at %s", ref, ulid, at)
		}
		res := showResultOf(ld.cfg, it)
		res.Skew = s.skew(cfg, ref, ulid, at)
		return &res, nil
	}
	it, err := storage.ReadItem(s.itemPath(ulid))
	if err != nil {
		return nil, errx.User("reading %s: %v", ref, err)
	}
	res := showResultOf(cfg, it)
	res.StderrNotes = ld.notes

	events, links, err := index.LogEntries(s.fs(), ulid, s.fileHead(ulid), func() ([]datamodel.Event, error) {
		return s.deriveEvents(ulid)
	})
	if err != nil {
		return nil, err
	}
	res.LinkedCommits = linkedCommitsView(links)
	res.HistoryTail = historyTailView(events)
	return &res, nil
}

func linkedCommitsView(links []index.CommitLink) []datamodel.CommitLink {
	out := make([]datamodel.CommitLink, len(links))
	for i, l := range links {
		out[i] = datamodel.CommitLink{SHA: l.SHA, Subject: l.Subject, Author: l.Author, Ts: l.Ts}
	}
	return out
}

func historyTailView(events []datamodel.Event) []datamodel.HistoryEvent {
	if len(events) > historyTailLimit {
		events = events[:historyTailLimit]
	}
	out := make([]datamodel.HistoryEvent, len(events))
	for i, e := range events {
		out[i] = datamodel.HistoryEvent{Ts: e.Ts, Field: e.Field, From: strOrNil(e.Old), To: strOrNil(e.New)}
	}
	return out
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
