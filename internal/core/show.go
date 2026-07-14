package core

import (
	"github.com/shivamshivanshu/kira/internal/codec"
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
		return nil, errx.User("reading %s from %s: %v", ref, s.itemPath(ulid), err).
			WithHint("the file is malformed; repair it in a text editor, then verify with `kira doctor`")
	}
	res := showResultOf(cfg, it)
	res.StderrNotes = ld.notes

	events, links, err := s.logEntries(ulid)
	if err != nil {
		return nil, err
	}
	res.LinkedCommits = commitLinksView(links, index.LinkLinked)
	res.ReferencedBy = commitLinksView(links, index.LinkReferenced)
	res.HistoryTail = historyTailView(events)
	return &res, nil
}

func commitLinksView(links []index.CommitLink, kind index.LinkKind) []datamodel.CommitLink {
	out := []datamodel.CommitLink{}
	for _, l := range links {
		if l.Kind == kind {
			out = append(out, datamodel.CommitLink{SHA: l.SHA, Subject: l.Subject, Author: l.Author, Ts: l.Ts})
		}
	}
	return out
}

func historyTailView(events []datamodel.Event) []datamodel.HistoryEvent {
	if len(events) > historyTailLimit {
		events = events[:historyTailLimit]
	}
	out := make([]datamodel.HistoryEvent, len(events))
	for i, e := range events {
		out[i] = datamodel.HistoryEvent{Ts: e.Ts, Field: e.Field, From: ptrOrNil(e.Old), To: ptrOrNil(e.New)}
	}
	return out
}

func showResultOf(cfg *datamodel.Config, it *datamodel.Item) datamodel.ShowResult {
	comments := codec.ParseComments(it.Body)
	views := make([]datamodel.CommentView, len(comments))
	for i, c := range comments {
		views[i] = datamodel.CommentView{ID: c.ID, Author: c.Author, Ts: c.Ts, Text: c.Body}
	}
	return datamodel.ShowResult{
		ID:            it.ID,
		Number:        it.Number,
		Board:         boardKeyOf(it.Number),
		Aliases:       nonNil(it.Aliases),
		Type:          it.Type,
		Subtype:       it.Subtype,
		Title:         it.Title,
		State:         it.State,
		Category:      categoryString(cfg, it.Type, it.State),
		Resolution:    it.Resolution,
		Priority:      it.Priority,
		Rank:          it.Rank,
		Sprint:        it.Sprint,
		Due:           it.Due,
		Owner:         it.Owner,
		Reporter:      it.Reporter,
		Labels:        nonNil(it.Labels),
		Epic:          it.Epic,
		BlockedBy:     nonNil(it.BlockedBy),
		Links:         linksView(it.Links),
		Blocks:        []string{},
		Estimate:      it.Estimate,
		Created:       it.Created,
		Updated:       it.Updated,
		Body:          it.Body,
		Comments:      views,
		LinkedCommits: []datamodel.CommitLink{},
		ReferencedBy:  []datamodel.CommitLink{},
		HistoryTail:   []datamodel.HistoryEvent{},
	}
}

func linksView(links map[string][]string) map[string][]string {
	view := make(map[string][]string, len(datamodel.LinkTypes))
	for _, typ := range datamodel.LinkTypes {
		view[string(typ)] = nonNil(links[string(typ)])
	}
	return view
}
