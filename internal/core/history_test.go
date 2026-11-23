package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

// itemMetric is the single-item form of itemsMetrics, for tests that assert on
// one ticket's derived metrics.
func itemMetric(s *Store, it *datamodel.Item) (metricItem, error) {
	mis, err := s.itemsMetrics(nil, []*datamodel.Item{it})
	if err != nil {
		return metricItem{}, err
	}
	return mis[0], nil
}

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
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")
	commitState(t, s, it, "IN_PROGRESS", "2026-01-06")
	commitState(t, s, it, "REVIEW", "2026-01-07")
	commitState(t, s, it, "DONE", "2026-01-08")

	di, err := itemMetric(s, it)
	if err != nil {
		t.Fatal(err)
	}
	if di.degraded {
		t.Error("fully observed history flagged degraded")
	}
	if want := time.Date(2026, 1, 8, 10, 0, 0, 0, time.UTC); !di.doneAt.Equal(want) {
		t.Errorf("doneAt = %v, want %v", di.doneAt, want)
	}
}

func TestItemMetricsCreatedAlreadyDone(t *testing.T) {
	s := eventRepo(t)
	it := eventTicket()
	commitState(t, s, it, "DONE", "2026-01-05")

	di, err := itemMetric(s, it)
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
	it := eventTicket()
	it.State = "DONE"
	it.Updated = "2026-01-09T10:00:00Z"
	if _, err := s.fs().WriteItem(it); err != nil {
		t.Fatal(err)
	}

	di, err := itemMetric(s, it)
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

func TestConfiguredMetricsReplayCommittedSubtypeAndState(t *testing.T) {
	s := eventRepo(t)
	it := eventTicket()
	cfg := &datamodel.Config{TicketStates: map[string][]datamodel.State{
		"task": {
			{Key: "[]", Category: datamodel.CategoryTodo},
			{Key: "DOING", Category: datamodel.CategoryDoing},
			{Key: "REVIEWING", Category: datamodel.CategoryDoing},
			{Key: "CLOSED", Category: datamodel.CategoryDone},
		},
		"bug": {
			{Key: "[]", Category: datamodel.CategoryDone},
			{Key: "DOING", Category: datamodel.CategoryTodo},
			{Key: "CLOSED", Category: datamodel.CategoryDone},
		},
	}}
	task, bug := "task", "bug"
	it.Subtype = &task
	commitState(t, s, it, "[]", "2026-01-05")
	it.Subtype = &bug
	commitState(t, s, it, "[]", "2026-01-06")
	it.Subtype = &task
	commitState(t, s, it, "DOING", "2026-01-07")
	commitState(t, s, it, "REVIEWING", "2026-01-08")

	it.Subtype = &bug
	it.State = "CLOSED"
	if _, err := s.fs().WriteItem(it); err != nil {
		t.Fatal(err)
	}
	wantDone := time.Date(2026, 1, 6, 10, 0, 0, 0, time.UTC)
	wantDoing := time.Date(2026, 1, 7, 10, 0, 0, 0, time.UTC)
	for range 2 {
		metrics, err := s.itemsMetrics(cfg, []*datamodel.Item{it})
		if err != nil {
			t.Fatal(err)
		}
		got := metrics[0]
		if got.degraded || got.category != datamodel.CategoryDone || !got.hasDoing || !got.hasDone ||
			got.reopens != 1 || !got.doneAt.Equal(wantDone) || !got.doingAt.Equal(wantDoing) {
			t.Errorf("metrics = %+v, want live done, first close %s, first doing %s and one reopen", got, wantDone, wantDoing)
		}
	}
}

func TestConfiguredMetricsCountCommittedInitialCategoryAtCreation(t *testing.T) {
	cfg := &datamodel.Config{TicketStates: map[string][]datamodel.State{
		"task": {
			{Key: "DOING", Category: datamodel.CategoryDoing},
			{Key: "CLOSED", Category: datamodel.CategoryDone},
		},
	}}
	for _, init := range []datamodel.State{
		{Key: "DOING", Category: datamodel.CategoryDoing},
		{Key: "CLOSED", Category: datamodel.CategoryDone},
	} {
		t.Run(init.Key, func(t *testing.T) {
			s := eventRepo(t)
			it := eventTicket()
			task := "task"
			it.Subtype = &task
			commitState(t, s, it, init.Key, "2026-01-05")
			got, err := s.itemsMetrics(cfg, []*datamodel.Item{it})
			if err != nil {
				t.Fatal(err)
			}
			metric := got[0]
			want := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC)
			switch init.Category {
			case datamodel.CategoryDoing:
				if !metric.hasDoing || !metric.doingAt.Equal(want) || metric.hasDone {
					t.Errorf("metrics = %+v, want doing at %s", metric, want)
				}
			case datamodel.CategoryDone:
				if !metric.hasDone || !metric.doneAt.Equal(want) || metric.hasDoing {
					t.Errorf("metrics = %+v, want done at %s", metric, want)
				}
			}
		})
	}
}

func TestConfiguredMetricsDoNotInferHistoryFromUnknownOrDirtyState(t *testing.T) {
	s := eventRepo(t)
	it := eventTicket()
	task := "task"
	it.Subtype = &task
	cfg := &datamodel.Config{TicketStates: map[string][]datamodel.State{
		task: {
			{Key: "DOING", Category: datamodel.CategoryDoing},
			{Key: "CLOSED", Category: datamodel.CategoryDone},
		},
	}}
	commitState(t, s, it, "CLOSED", "2026-01-05")
	commitState(t, s, it, "RETIRED", "2026-01-06")
	commitState(t, s, it, "DOING", "2026-01-07")
	got, err := s.itemsMetrics(cfg, []*datamodel.Item{it})
	if err != nil {
		t.Fatal(err)
	}
	if metric := got[0]; !metric.degraded || !metric.hasDone || metric.hasDoing || metric.reopens != 0 {
		t.Errorf("metrics = %+v, want known initial close and no inferred reopen", metric)
	}

	it.State = "CLOSED"
	if metric := metricsFromWorkflow(cfg, it, nil, nil, false); !metric.degraded || metric.hasDone || metric.category != datamodel.CategoryDone {
		t.Errorf("metrics = %+v, want live done and no synthesized history", metric)
	}
}

func TestMetricsAcrossTicketPromotion(t *testing.T) {
	task := "task"
	custom := &datamodel.Config{TicketStates: map[string][]datamodel.State{
		task: {
			{Key: "NEW", Category: datamodel.CategoryTodo},
			{Key: "WIP", Category: datamodel.CategoryDoing},
			{Key: "CHECK", Category: datamodel.CategoryDoing},
			{Key: "CLOSED", Category: datamodel.CategoryDone},
		},
	}}
	for _, tc := range []struct {
		name          string
		cfg           *datamodel.Config
		ticketStates  []string
		epicState     string
		firstDoingDay int
		firstDoneDay  int
		reopens       int
		closeEpic     bool
	}{
		{name: "active promotion then close", ticketStates: []string{"TODO", "IN_PROGRESS"}, epicState: "ACTIVE", firstDoingDay: 6, closeEpic: true},
		{name: "type only", ticketStates: []string{"TODO", "IN_PROGRESS", "DONE"}, epicState: "DONE", firstDoingDay: 6, firstDoneDay: 7},
		{name: "created done", ticketStates: []string{"DONE"}, epicState: "DONE", firstDoneDay: 5},
		{name: "built-in reopens", ticketStates: []string{"TODO", "DONE", "IN_PROGRESS", "REVIEW"}, epicState: "ACTIVE", firstDoingDay: 7, firstDoneDay: 6, reopens: 2},
		{name: "configured reopen", cfg: custom, ticketStates: []string{"NEW", "WIP", "CLOSED", "WIP", "CHECK"}, epicState: "ACTIVE", firstDoingDay: 6, firstDoneDay: 7, reopens: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := eventRepo(t)
			it := eventTicket()
			if tc.cfg != nil {
				it.Subtype = &task
			}
			for i, state := range tc.ticketStates {
				commitState(t, s, it, state, promotionMetricTime(5+i).Format(time.DateOnly))
			}
			assertMetrics := func(phase string, doneDay int) {
				t.Helper()
				got, err := s.itemsMetrics(tc.cfg, []*datamodel.Item{it})
				if err != nil {
					t.Fatal(err)
				}
				category, _ := tc.cfg.ItemCategory(it)
				want := metricItem{
					number: it.Number, category: category, created: promotionMetricTime(5),
					hasDoing: tc.firstDoingDay != 0, doingAt: promotionMetricTime(tc.firstDoingDay),
					hasDone: doneDay != 0, doneAt: promotionMetricTime(doneDay), reopens: tc.reopens,
				}
				if got[0] != want {
					t.Fatalf("%s metrics = %+v, want %+v", phase, got[0], want)
				}
			}
			assertMetrics("ticket", tc.firstDoneDay)
			it.Type, it.State, it.Subtype = datamodel.TypeEpic, tc.epicState, nil
			promotionDate := promotionMetricTime(10).Format(time.DateOnly)
			it.Updated = promotionMetricTime(10).Format(time.RFC3339)
			path, err := s.fs().WriteItem(it)
			if err != nil {
				t.Fatal(err)
			}
			assertMetrics("unstaged", tc.firstDoneDay)
			gitRun(t, s, it.Updated, "add", "--", path)
			assertMetrics("staged", tc.firstDoneDay)
			commitState(t, s, it, tc.epicState, promotionDate)
			assertMetrics("committed with ticket cache primed", tc.firstDoneDay)
			if err := os.RemoveAll(s.fs().CacheDir()); err != nil {
				t.Fatal(err)
			}
			assertMetrics("committed cold", tc.firstDoneDay)
			assertMetrics("committed warm", tc.firstDoneDay)
			if tc.closeEpic {
				commitState(t, s, it, "DONE", promotionMetricTime(11).Format(time.DateOnly))
				assertMetrics("closed epic", 11)
			}
		})
	}
}

func promotionMetricTime(day int) time.Time {
	if day == 0 {
		return time.Time{}
	}
	return time.Date(2026, 1, day, 10, 0, 0, 0, time.UTC)
}

func TestRecreatedEpicRetainsMetrics(t *testing.T) {
	s := eventRepo(t)
	it := eventTicket()
	it.Type = datamodel.TypeEpic
	commitState(t, s, it, "PLANNED", "2026-01-05")
	commitState(t, s, it, "ACTIVE", "2026-01-06")
	commitState(t, s, it, "DONE", "2026-01-07")
	before, err := itemMetric(s, it)
	if err != nil {
		t.Fatal(err)
	}
	if !before.hasDoing || !before.hasDone {
		t.Fatalf("expected active and done history, got %+v", before)
	}

	gitRun(t, s, "2026-01-08T10:00:00Z", "rm", "--", s.fs().RelToRoot(s.itemPath(it.ID)))
	gitRun(t, s, "2026-01-08T10:00:00Z", "commit", "-m", "remove epic")
	commitState(t, s, it, "DONE", "2026-01-09")
	after, err := itemMetric(s, it)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("metrics after restoring epic = %+v, want %+v", after, before)
	}
}
