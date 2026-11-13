package core

import (
	"os"
	"slices"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

const historyTailLimit = 10

// ItemDetail pairs displayed content with a revision of the same file bytes.
type ItemDetail struct {
	Item     *datamodel.ShowResult
	Revision string
	store    *Store
}

func (s *Store) Show(cfg *datamodel.Config, ref, at string) (*datamodel.ShowResult, error) {
	ld, err := s.read(cfg, loadOpts{at: at, useIndex: true})
	if err != nil {
		return nil, err
	}
	if at == "" {
		detail, err := s.detail(ld, ref)
		if err != nil {
			return nil, err
		}
		return detail.Item, nil
	}
	ulid, err := resolveID(ld.resolver, ref)
	if err != nil {
		return nil, err
	}
	it := findByULID(ld.items, ulid)
	if it == nil {
		return nil, errx.User("%s resolved to %s, which is absent at %s", ref, ulid, at)
	}
	res := crossReferencedShowResultOf(ld, it, ulid)
	res.Skew = s.skew(cfg, ref, ulid, at)
	return &res, nil
}

func (s *Store) detail(ld *loaded, ref string) (*ItemDetail, error) {
	ulid, err := resolveID(ld.resolver, ref)
	if err != nil {
		return nil, err
	}
	path := s.itemPath(ulid)
	data, err := os.ReadFile(path)
	var it *datamodel.Item
	if err == nil {
		it, err = codec.Parse(string(data))
	}
	if err != nil {
		return nil, errx.User("reading %s from %s: %v", ref, path, err).
			WithHint("the file is malformed; repair it in a text editor, then verify with `kira doctor`")
	}
	res := crossReferencedShowResultOf(ld, it, ulid)
	res.StderrNotes = ld.notes

	events, links, err := s.logEntries(ulid)
	if err != nil {
		return nil, err
	}
	res.LinkedCommits = commitLinksView(links, index.LinkLinked)
	res.ReferencedBy = commitLinksView(links, index.LinkReferenced)
	res.HistoryTail = historyTailView(events)
	return &ItemDetail{Item: &res, Revision: ItemRevision(string(data)), store: s}, nil
}

// blocksOf derives the reverse of blocked_by: the items that name ulid as a
// blocker. It is not stored on the item itself, so it is recomputed from the
// loaded set rather than read from the file.
// numberReferences fills in the display number for every ULID the result
// points at. internal/id keeps ULIDs internal and numbers for humans, but
// ShowResult carried only the ULIDs, so kira show printed them raw.
func numberReferences(res *datamodel.ShowResult, items []*datamodel.Item) {
	byULID := datamodel.IndexByID(items)
	numberOf := func(ulid string) (string, bool) {
		it, ok := byULID[ulid]
		if !ok {
			return "", false
		}
		return it.Number, true
	}
	if res.Parent != nil {
		if n, ok := numberOf(*res.Parent); ok {
			res.ParentNumber = &n
		}
	}
	res.BlockedByNumbers = numbersFor(numberOf, res.BlockedBy)
	res.BlocksNumbers = numbersFor(numberOf, res.Blocks)
	if len(res.Links) > 0 {
		res.LinksNumbers = make(map[string][]string, len(res.Links))
		for typ, targets := range res.Links {
			res.LinksNumbers[typ] = numbersFor(numberOf, targets)
		}
	}
}

// numbersFor keeps the ULID where it resolves to nothing: a dangling reference
// should stay visible, not render as a gap.
func numbersFor(numberOf func(string) (string, bool), ulids []string) []string {
	out := make([]string, len(ulids))
	for i, u := range ulids {
		if n, ok := numberOf(u); ok {
			out[i] = n
			continue
		}
		out[i] = u
	}
	return out
}

func blocksOf(items []*datamodel.Item, ulid string) []string {
	out := []string{}
	for _, it := range items {
		if slices.Contains(it.BlockedBy, ulid) {
			out = append(out, it.ID)
		}
	}
	slices.Sort(out)
	return out
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
		out[i] = datamodel.HistoryEvent{Ts: e.Ts, Field: e.Field, From: ptr.NilIfEmpty(e.Old), To: ptr.NilIfEmpty(e.New)}
	}
	return out
}

// These fields depend on the whole loaded set, not on the item alone, which is
// what separates this from showResultOf.
func crossReferencedShowResultOf(ld *loaded, it *datamodel.Item, ulid string) datamodel.ShowResult {
	res := showResultOf(it)
	res.Blocks = blocksOf(ld.items, ulid)
	numberReferences(&res, ld.items)
	return res
}

func showResultOf(it *datamodel.Item) datamodel.ShowResult {
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
		Category:      categoryString(it.Type, it.State),
		Resolution:    it.Resolution,
		Priority:      it.Priority,
		Rank:          it.Rank,
		Sprint:        it.Sprint,
		Due:           it.Due,
		Owner:         it.Owner,
		Reporter:      it.Reporter,
		Labels:        nonNil(it.Labels),
		Parent:        it.Parent,
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
