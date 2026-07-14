package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func eventRepo(t *testing.T) *Store {
	t.Helper()
	dir := testutil.InitGitRepo(t)
	if err := os.MkdirAll(filepath.Join(dir, ".kira", "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	return newStore(dir)
}

func commitState(t *testing.T, s *Store, it *datamodel.Item, state, date string) {
	t.Helper()
	it.State = state
	it.Updated = date + "T10:00:00Z"
	if _, err := s.fs().WriteItem(it); err != nil {
		t.Fatal(err)
	}
	stamp := date + "T10:00:00Z"
	gitRun(t, s, stamp, "add", "-A")
	gitRun(t, s, stamp, "commit", "-m", "state "+state)
}

func eventTicket() *datamodel.Item {
	return &datamodel.Item{
		ID:      "01HZZ0TEST0000000000000000",
		Number:  "KIRA-1",
		Type:    "ticket",
		Title:   "T",
		State:   "TODO",
		Labels:  []string{},
		Created: "2026-01-05T10:00:00Z",
		Updated: "2026-01-05T10:00:00Z",
	}
}

func TestItemMetricsCleanHistory(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")
	commitState(t, s, it, "IN_PROGRESS", "2026-01-06")
	commitState(t, s, it, "REVIEW", "2026-01-07")
	commitState(t, s, it, "DONE", "2026-01-08")

	di, err := s.itemMetrics(cfg, it, s.fileHead(it.ID))
	if err != nil {
		t.Fatal(err)
	}
	if di.degraded {
		t.Error("on-graph history flagged degraded")
	}
	if want := time.Date(2026, 1, 8, 10, 0, 0, 0, time.UTC); !di.doneAt.Equal(want) {
		t.Errorf("doneAt = %v, want %v", di.doneAt, want)
	}
}

func TestItemMetricsSquashedHistoryDegrades(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")
	commitState(t, s, it, "DONE", "2026-01-08")

	di, err := s.itemMetrics(cfg, it, s.fileHead(it.ID))
	if err != nil {
		t.Fatal(err)
	}
	if !di.degraded {
		t.Error("off-graph jump not flagged degraded")
	}
	if !di.hasDone {
		t.Error("hasDone false, want done recorded from the squash commit")
	}
}

func TestItemMetricsCreatedAlreadyDone(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	commitState(t, s, it, "DONE", "2026-01-05")

	di, err := s.itemMetrics(cfg, it, s.fileHead(it.ID))
	if err != nil {
		t.Fatal(err)
	}
	if di.degraded {
		t.Error("created-done item flagged degraded")
	}
	if want := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC); !di.doneAt.Equal(want) {
		t.Errorf("doneAt = %v, want %v", di.doneAt, want)
	}
}

func TestItemMetricsUncommittedFallsBackToUpdated(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	it.State = "DONE"
	it.Updated = "2026-01-09T10:00:00Z"
	if _, err := s.fs().WriteItem(it); err != nil {
		t.Fatal(err)
	}

	di, err := s.itemMetrics(cfg, it, s.fileHead(it.ID))
	if err != nil {
		t.Fatal(err)
	}
	if !di.degraded || !di.hasDone {
		t.Errorf("doneInfo = %+v, want degraded fallback to updated", di)
	}
	upd, _ := time.Parse(time.RFC3339, it.Updated)
	if !di.doneAt.Equal(upd) {
		t.Errorf("doneAt = %v, want %v", di.doneAt, upd)
	}
}

func TestCachedStateEventsChronology(t *testing.T) {
	s := eventRepo(t)
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")
	commitState(t, s, it, "IN_PROGRESS", "2026-01-06")
	commitState(t, s, it, "REVIEW", "2026-01-07")

	evs, committed, err := s.cachedStateEvents(it.ID, s.fileHead(it.ID))
	if err != nil {
		t.Fatal(err)
	}
	if !committed {
		t.Error("committed = false, want true")
	}
	if len(evs) != 2 {
		t.Fatalf("events = %+v, want 2 (creation is not a transition event)", evs)
	}
	if evs[0].from != "TODO" || evs[0].to != "IN_PROGRESS" {
		t.Errorf("first event = %+v, want TODO -> IN_PROGRESS", evs[0])
	}
	if evs[1].from != "IN_PROGRESS" || evs[1].to != "REVIEW" {
		t.Errorf("second event = %+v, want IN_PROGRESS -> REVIEW", evs[1])
	}
	if !strings.HasPrefix(evs[0].ts.UTC().Format(time.RFC3339), "2026-01-06") {
		t.Errorf("first event ts = %v, want 2026-01-06", evs[0].ts)
	}
}
