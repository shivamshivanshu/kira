package core

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/setx"
	"github.com/shivamshivanshu/kira/internal/treeish"
)

func (s *Store) Review(base string) (*datamodel.ReviewResult, error) {
	repo := s.repo()
	baseSHA, err := repo.ResolveTreeish(base)
	if err != nil {
		return nil, errx.User("resolving --base %s: %v", base, err)
	}
	head, err := repo.ResolveTreeish("HEAD")
	if err != nil {
		return nil, errx.User("resolving HEAD: %v", err)
	}
	mergeBase, err := repo.MergeBase(baseSHA, head)
	if err != nil {
		return nil, errx.User("finding merge base for %s and HEAD: %v", base, err)
	}
	from, err := treeish.LoadCommit(repo, mergeBase)
	if err != nil {
		return nil, err
	}
	to, err := treeish.LoadCommit(repo, head)
	if err != nil {
		return nil, err
	}
	before, after := datamodel.IndexByID(from.Items), datamodel.IndexByID(to.Items)
	events, err := s.rangeEvents(repo, baseSHA, head, after)
	if err != nil {
		return nil, err
	}
	commitRange := baseSHA + ".." + head
	commits, err := repo.LogTrailers(commitRange, datamodel.TicketTrailerKey, datamodel.CloseTrailerKey)
	if err != nil {
		return nil, err
	}
	numbers := reviewNumbers(to)
	linked, unresolved := reviewLinks(commits, to, after, numbers)
	landedRef := s.landedRef(to.Config)
	landedHead := ""
	switch landedRef {
	case "HEAD":
		landedHead = head
	case base:
		landedHead = baseSHA
	default:
		landedHead, _ = repo.ResolveTreeish(landedRef)
	}
	closes, err := reviewCloses(repo, to.Config, landedHead, commitRange, commits, after, numbers)
	if err != nil {
		return nil, err
	}
	changes, history := reviewStates(before, after, events)
	return &datamodel.ReviewResult{
		Base: baseSHA, Head: head, MergeBase: mergeBase, LandedRef: landedRef,
		Linked: linked, Unresolved: unresolved, StateChanges: changes, StateEvents: history, Closes: closes,
		StderrNotes: mergedWarnings(from, to),
	}, nil
}

func reviewTicket(it *datamodel.Item) datamodel.ReviewTicket {
	return datamodel.ReviewTicket{ID: it.ID, Number: it.Number, Title: it.Title}
}

func reviewTicketKey(it datamodel.ReviewTicket) id.SortKey {
	return id.NewSortKey(it.Number, it.ID)
}

func reviewStates(before, after map[string]*datamodel.Item, events map[string][]datamodel.Event) ([]datamodel.ReviewStateChange, []datamodel.ReviewStateHistory) {
	changes := []datamodel.ReviewStateChange{}
	history := []datamodel.ReviewStateHistory{}
	for uid, it := range after {
		ticket := reviewTicket(it)
		if prior := before[uid]; prior != nil && prior.State != it.State {
			changes = append(changes, datamodel.ReviewStateChange{Ticket: ticket, From: prior.State, To: it.State})
		}
		var states []datamodel.Event
		for _, event := range events[uid] {
			if event.Field == datamodel.KeyState {
				states = append(states, event)
			}
		}
		if len(states) > 0 {
			history = append(history, datamodel.ReviewStateHistory{Ticket: ticket, Events: states})
		}
	}
	sortByKey(changes, func(change datamodel.ReviewStateChange) id.SortKey { return reviewTicketKey(change.Ticket) })
	sortByKey(history, func(entry datamodel.ReviewStateHistory) id.SortKey { return reviewTicketKey(entry.Ticket) })
	return changes, history
}

func reviewNumbers(snapshot *treeish.Loaded) map[string]string {
	numbers := make(map[string]string, len(snapshot.Items))
	storedIDs := make(map[string]string, len(snapshot.Items))
	for _, it := range snapshot.Items {
		normalized := strings.ToUpper(it.ID)
		if previous, exists := storedIDs[normalized]; exists && previous != it.ID {
			storedIDs[normalized] = ""
			continue
		}
		storedIDs[normalized] = it.ID
	}
	for _, it := range snapshot.Items {
		for _, number := range append([]string{it.Number}, it.Aliases...) {
			key := strings.ToUpper(number)
			if _, seen := numbers[key]; !seen {
				uid, _ := snapshot.Resolver.LookupNumber(key)
				numbers[key] = storedIDs[uid]
			}
		}
	}
	return numbers
}

func reviewLinks(commits []gitx.Commit, snapshot *treeish.Loaded, items map[string]*datamodel.Item, numbers map[string]string) ([]datamodel.ReviewLink, []datamodel.ReviewUnresolved) {
	byID := make(map[string][]datamodel.ReviewCommit)
	unresolved := []datamodel.ReviewUnresolved{}
	policy := index.NewLinkPolicy(indexOptions(snapshot.Config), numbers)
	for _, commit := range commits {
		seen := setx.NewDeduper[string]()
		seenUnresolved := setx.NewDeduper[datamodel.ReviewUnresolved]()
		add := func(token, source string) bool {
			uid := numbers[strings.ToUpper(token)]
			if it := items[uid]; it != nil {
				if seen.Add(uid) {
					byID[uid] = append(byID[uid], datamodel.ReviewCommit{SHA: commit.SHA, Subject: commit.Subject})
				}
				return true
			}
			resolved, ambiguous := snapshot.Resolver.LookupNumber(token)
			missing := datamodel.ReviewUnresolved{Ref: token, Source: source, CommitSHA: commit.SHA, Ambiguous: ambiguous || resolved != ""}
			if seenUnresolved.Add(missing) {
				unresolved = append(unresolved, missing)
			}
			return false
		}
		policy.VisitLinkedRefs(commit, add)
		for _, token := range commit.Closes {
			add(token, datamodel.CloseTrailerKey)
		}
	}
	linked := make([]datamodel.ReviewLink, 0, len(byID))
	for uid, links := range byID {
		linked = append(linked, datamodel.ReviewLink{Ticket: reviewTicket(items[uid]), Commits: links})
	}
	sortByKey(linked, func(link datamodel.ReviewLink) id.SortKey { return reviewTicketKey(link.Ticket) })
	return linked, unresolved
}

func reviewCloses(repo gitx.Repo, cfg *datamodel.Config, landedHead, commitRange string, commits []gitx.Commit, items map[string]*datamodel.Item, numbers map[string]string) ([]datamodel.ReviewClose, error) {
	candidates, _ := index.LatestCloses(commits, numbers)
	closes := make([]datamodel.ReviewClose, 0, len(candidates))
	if len(candidates) == 0 {
		return closes, nil
	}
	unlanded, err := reviewUnlanded(repo, landedHead, commitRange)
	if err != nil {
		return nil, err
	}
	pending := pendingReviewCloses(commits, numbers, unlanded)
	for _, candidate := range candidates {
		it := items[candidate.ULID]
		if it == nil {
			continue
		}
		preview := datamodel.ReviewClose{Ticket: reviewTicket(it), CommitSHA: candidate.CommitSHA, Outcome: datamodel.ReviewCloseAlreadyLanded}
		selected, found := candidate, true
		if unlanded != nil {
			selected, found = pending[it.ID]
		}
		if found {
			preview.CommitSHA = selected.CommitSHA
			preview.Target, preview.Outcome, _ = closeEligibility(cfg, it, selected.CommitterTs)
		}
		closes = append(closes, preview)
	}
	sortByKey(closes, func(preview datamodel.ReviewClose) id.SortKey { return reviewTicketKey(preview.Ticket) })
	return closes, nil
}

func pendingReviewCloses(commits []gitx.Commit, numbers map[string]string, unlanded map[string]bool) map[string]index.CloseCandidate {
	if unlanded == nil {
		return nil
	}
	var pending []gitx.Commit
	for _, commit := range commits {
		if unlanded[commit.SHA] && len(commit.Closes) > 0 {
			pending = append(pending, commit)
		}
	}
	candidates, _ := index.LatestCloses(pending, numbers)
	out := make(map[string]index.CloseCandidate, len(candidates))
	for _, candidate := range candidates {
		out[candidate.ULID] = candidate
	}
	return out
}

func reviewUnlanded(repo gitx.Repo, landedHead, commitRange string) (map[string]bool, error) {
	if landedHead == "" {
		return nil, nil
	}
	shas, err := repo.Output("rev-list", commitRange, "^"+landedHead)
	if err != nil {
		return nil, err
	}
	return setx.ToSet(strings.Fields(shas)), nil
}
