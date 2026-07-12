package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/item"
)

func eventRepo(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e.c",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e.c",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	run("init")
	if err := os.MkdirAll(filepath.Join(dir, ".kira", "tickets"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &Store{root: dir}
}

func commitState(t *testing.T, s *Store, it *item.Item, state, date string) {
	t.Helper()
	it.State = state
	it.Updated = date + "T10:00:00Z"
	if _, err := s.writeItem(it); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-m", "state " + state}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = s.root
		cmd.Env = append(os.Environ(),
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e.c",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e.c",
			"GIT_AUTHOR_DATE="+date+"T10:00:00Z",
			"GIT_COMMITTER_DATE="+date+"T10:00:00Z",
			"TZ=UTC",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

func eventTicket() *item.Item {
	return &item.Item{
		ID:      "01TESTULID0000000000000000",
		Number:  "KIRA-1",
		Type:    "ticket",
		Title:   "T",
		State:   "TODO",
		Labels:  []string{},
		Created: "2026-01-05T10:00:00Z",
		Updated: "2026-01-05T10:00:00Z",
	}
}

func TestItemDoneInfoCleanHistory(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")
	commitState(t, s, it, "IN_PROGRESS", "2026-01-06")
	commitState(t, s, it, "REVIEW", "2026-01-07")
	commitState(t, s, it, "DONE", "2026-01-08")

	di, err := s.itemDoneInfo(cfg, it)
	if err != nil {
		t.Fatal(err)
	}
	if di.degraded {
		t.Error("on-graph history flagged degraded")
	}
	if want := time.Date(2026, 1, 8, 10, 0, 0, 0, time.UTC).Local().Format(time.DateOnly); di.doneDay != want {
		t.Errorf("doneDay = %q, want %q", di.doneDay, want)
	}
}

func TestItemDoneInfoSquashedHistoryDegrades(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")
	commitState(t, s, it, "DONE", "2026-01-08")

	di, err := s.itemDoneInfo(cfg, it)
	if err != nil {
		t.Fatal(err)
	}
	if !di.degraded {
		t.Error("off-graph jump not flagged degraded")
	}
	if di.doneDay == "" {
		t.Error("doneDay empty, want the squash commit's day")
	}
}

func TestItemDoneInfoCreatedAlreadyDone(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	commitState(t, s, it, "DONE", "2026-01-05")

	di, err := s.itemDoneInfo(cfg, it)
	if err != nil {
		t.Fatal(err)
	}
	if di.degraded {
		t.Error("created-done item flagged degraded")
	}
	if want := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC).Local().Format(time.DateOnly); di.doneDay != want {
		t.Errorf("doneDay = %q, want %q", di.doneDay, want)
	}
}

func TestItemDoneInfoUncommittedFallsBackToUpdated(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	it.State = "DONE"
	it.Updated = "2026-01-09T10:00:00Z"
	if _, err := s.writeItem(it); err != nil {
		t.Fatal(err)
	}

	di, err := s.itemDoneInfo(cfg, it)
	if err != nil {
		t.Fatal(err)
	}
	if !di.degraded || di.doneDay == "" {
		t.Errorf("doneInfo = %+v, want degraded fallback to updated day", di)
	}
	upd, _ := time.Parse(time.RFC3339, it.Updated)
	if want := upd.Local().Format(time.DateOnly); di.doneDay != want {
		t.Errorf("doneDay = %q, want %q", di.doneDay, want)
	}
}

func TestStateEventsChronology(t *testing.T) {
	s := eventRepo(t)
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")
	commitState(t, s, it, "IN_PROGRESS", "2026-01-06")

	evs, err := s.stateEvents(it.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("events = %+v, want 2", evs)
	}
	if evs[0].from != "" || evs[0].to != "TODO" {
		t.Errorf("creation event = %+v, want \"\" -> TODO", evs[0])
	}
	if evs[1].from != "TODO" || evs[1].to != "IN_PROGRESS" {
		t.Errorf("second event = %+v, want TODO -> IN_PROGRESS", evs[1])
	}
	if !strings.HasPrefix(evs[1].ts.UTC().Format(time.RFC3339), "2026-01-06") {
		t.Errorf("second event ts = %v, want 2026-01-06", evs[1].ts)
	}
}
