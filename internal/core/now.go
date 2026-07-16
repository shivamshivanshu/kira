package core

import (
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/workon"
)

func (s *Store) Now(cfg *datamodel.Config) (*datamodel.NowResult, error) {
	if err := s.requireRepo(); err != nil {
		return nil, err
	}
	ld, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		return nil, err
	}
	branch, _ := s.repo().CurrentBranch()
	it, activeBranch, source := s.activeItem(cfg, ld, branch)
	if it == nil {
		return nil, errx.User("no active ticket").WithHint("start one with `kira workon <id>`")
	}

	since := s.stateSince(it)
	byID := byULID(ld.items)
	blockers := make([]datamodel.NowBlocker, 0, len(it.BlockedBy))
	for _, b := range it.BlockedBy {
		blocker := byID[b]
		blockers = append(blockers, datamodel.NowBlocker{Number: numberOrID(blocker, b), State: blockerState(blocker)})
	}

	rev := activeBranch
	if rev == "" {
		rev = "HEAD"
	}
	commits, _ := s.repo().RevListSince(rev, sinceExclusive(since))

	category := categoryString(cfg, it.Type, it.State)
	return &datamodel.NowResult{
		ID:         it.ID,
		Number:     it.Number,
		Title:      it.Title,
		State:      it.State,
		Category:   category,
		StateSince: since,
		Due:        it.Due,
		Overdue:    datamodel.IsOverdue(it.Due, category, time.Now()),
		Branch:     activeBranch,
		Source:     source,
		Blockers:   blockers,
		Commits:    commitsView(commits),
	}, nil
}

func (s *Store) activeItem(cfg *datamodel.Config, ld *loaded, branch string) (*datamodel.Item, string, datamodel.NowSource) {
	if p, ok := s.readActive(); ok {
		if it := findByULID(ld.items, p.Ticket); it != nil {
			active := p.Branch
			if active == "" {
				active = branch
			}
			return it, active, datamodel.NowSourcePointer
		}
	}
	if number, ok := workon.InferNumber(branch, cfg.BoardKeys()); ok {
		if ulid, err := resolveID(ld.resolver, number); err == nil {
			if it := findByULID(ld.items, ulid); it != nil {
				return it, branch, datamodel.NowSourceBranch
			}
		}
	}
	return nil, "", ""
}

func (s *Store) stateSince(it *datamodel.Item) string {
	evs, _, err := s.cachedStateEvents(it.ID, s.fileHead(it.ID))
	if err == nil && len(evs) > 0 {
		return evs[len(evs)-1].ts.Format(time.RFC3339)
	}
	return it.Created
}

func sinceExclusive(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Add(time.Second).Format(time.RFC3339)
}

func blockerState(blocker *datamodel.Item) string {
	if blocker == nil {
		return ""
	}
	return blocker.State
}

func commitsView(commits []gitx.Commit) []datamodel.CommitLink {
	out := make([]datamodel.CommitLink, len(commits))
	for i, c := range commits {
		out[i] = datamodel.CommitLink{SHA: c.SHA, Subject: c.Subject, Author: c.Author, Ts: c.Timestamp}
	}
	return out
}
