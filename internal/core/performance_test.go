package core

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func checkBatchedHistory(t *testing.T, s *Store, ids ...string) map[string][]datamodel.Event {
	t.Helper()
	revision, err := s.repo().ResolveTreeish("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.deriveEventBatch(revision, ids)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		want, err := s.deriveEvents(id)
		if err != nil || !slices.Equal(got[id], want) {
			t.Fatalf("%s events %v != %v (%v)", id, got[id], want, err)
		}
	}
	return got
}

func TestBatchedHistoryMatchesPerFile(t *testing.T) {
	s := newStore(filepath.Join(eventRepo(t).root, "prøject"))
	a, b, resolved := eventTicket(), eventTicket(), eventTicket()
	b.ID, b.Number = "01HZZ0TEST0000000000000001", "KIRA-2"
	resolved.ID, resolved.Number = "01HZZ0TEST0000000000000002", "KIRA-3"
	commitState(t, s, a, "TODO", "2026-01-05")
	commitState(t, s, b, "TODO", "2026-01-05")
	commitState(t, s, resolved, "TODO", "2026-01-05")
	commitState(t, s, a, "IN_PROGRESS", "2026-01-06")
	commitState(t, s, b, "DONE", "2026-01-07")
	revision, err := s.repo().ResolveTreeish("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	heads := s.fileHeads(revision)
	for _, ticket := range []*datamodel.Item{a, b, resolved} {
		if want := s.fileHead(ticket.ID); want == "" || heads[ticket.ID] != want {
			t.Fatalf("%s file head = %q, want %q", ticket.Number, heads[ticket.ID], want)
		}
	}
	checkBatchedHistory(t, s, a.ID, b.ID, resolved.ID)

	gitRun(t, s, "", "checkout", "-b", "side")
	commitState(t, s, a, "REVIEW", "2026-01-08")
	commitState(t, s, b, "REVIEW", "2026-01-08")
	gitRun(t, s, "", "checkout", "main", "--", s.fs().RelToRoot(s.itemPath(b.ID)))
	gitRun(t, s, "2026-01-08T12:00:00Z", "commit", "-m", "restore ticket on side")
	commitState(t, s, resolved, "IN_PROGRESS", "2026-01-08")
	gitRun(t, s, "", "checkout", "main")
	commitState(t, s, b, "TODO", "2026-01-09")
	gitRun(t, s, "", "merge", "--no-ff", "--no-commit", "side")
	commitState(t, s, resolved, "DONE", "2026-01-10")
	checkBatchedHistory(t, s, a.ID, b.ID, resolved.ID)
}

func TestBatchedHistoryPreservesDeletion(t *testing.T) {
	s := eventRepo(t)
	a, b := eventTicket(), eventTicket()
	b.ID, b.Number = "01HZZ0TEST0000000000000001", "KIRA-2"
	commitState(t, s, a, "TODO", "2026-01-05")
	commitState(t, s, b, "TODO", "2026-01-06")
	gitRun(t, s, "", "rm", "--", s.fs().RelToRoot(s.itemPath(a.ID)))
	gitRun(t, s, "2026-01-07T10:00:00Z", "commit", "-m", "delete ticket")
	commitState(t, s, a, "DONE", "2026-01-08")
	got := checkBatchedHistory(t, s, a.ID, b.ID)
	if !slices.ContainsFunc(got[a.ID], func(e datamodel.Event) bool {
		return e.Field == datamodel.KeyState && e.Old == "TODO" && e.New == ""
	}) {
		t.Fatalf("missing deletion state event in %v", got[a.ID])
	}
}

func TestBatchedHistoryHonorsRenameFollowing(t *testing.T) {
	s := eventRepo(t)
	a, b := eventTicket(), eventTicket()
	b.ID, b.Number = "01HZZ0TEST0000000000000001", "KIRA-2"
	commitState(t, s, a, "TODO", "2026-01-05")
	commitState(t, s, b, "TODO", "2026-01-06")
	commitState(t, s, a, "IN_PROGRESS", "2026-01-07")
	renamedID := "01HZZ0TEST0000000000000002"
	gitRun(t, s, "", "mv", "--", s.fs().RelToRoot(s.itemPath(a.ID)), s.fs().RelToRoot(s.itemPath(renamedID)))
	gitRun(t, s, "2026-01-08T10:00:00Z", "commit", "-m", "rename ticket")
	gitRun(t, s, "", "config", "log.follow", "true")
	got := checkBatchedHistory(t, s, renamedID, b.ID)
	if !slices.ContainsFunc(got[renamedID], func(e datamodel.Event) bool {
		return e.Field == datamodel.KeyState && e.Old == "TODO" && e.New == "IN_PROGRESS"
	}) {
		t.Fatalf("missing pre-rename state event in %v", got[renamedID])
	}
}

func TestStatsWithMultipleUncommittedTickets(t *testing.T) {
	s := eventRepo(t)
	a, b := eventTicket(), eventTicket()
	b.ID, b.Number = "01HZZ0TEST0000000000000001", "KIRA-2"
	for _, item := range []*datamodel.Item{a, b} {
		if _, err := s.fs().WriteItem(item); err != nil {
			t.Fatal(err)
		}
	}
	gitRun(t, s, "", "add", "-A")
	stats, err := s.Stats(config.Default(), StatsOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Completion.Total != 2 {
		t.Fatalf("completion total = %d, want 2", stats.Completion.Total)
	}
}
