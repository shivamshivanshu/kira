package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func sampleStats() *datamodel.StatsResult {
	return &datamodel.StatsResult{
		Scope:      &datamodel.StatsScope{Sprint: "2026-S1", Weeks: 8},
		Completion: &datamodel.Completion{Done: 5, Total: 10, Dropped: 1, Pct: 0.5},
		CycleTime:  &datamodel.Percentiles{P50: 5, P90: 8.8, N: 4},
		LeadTime:   &datamodel.Percentiles{P50: 5, P90: 9.4, N: 5},
		Throughput: []int{0, 1, 3, 2, 0, 4, 1, 2},
		Reopens:    &datamodel.Reopens{Count: 3, Items: []string{"KIRA-3", "KIRA-4"}},
	}
}

func statsScreenWith(res *datamodel.StatsResult) (model, *statsScreen) {
	m := newTestModel(100, 20, false)
	ss := m.screens[viewStats].(*statsScreen)
	if res != nil {
		ss.res, ss.loaded = res, true
	}
	return m, ss
}

func TestStatsScreenRender(t *testing.T) {
	t.Parallel()
	m, ss := statsScreenWith(sampleStats())
	golden.RequireEqual(t, []byte(ss.view(&m, 100, 18)))
}

func TestStatsScreenEmptyState(t *testing.T) {
	t.Parallel()
	m, ss := statsScreenWith(nil)
	got := ss.view(&m, 100, 18)
	if !strings.Contains(got, "No metrics yet") {
		t.Fatalf("empty stats should teach; got:\n%s", got)
	}
	golden.RequireEqual(t, []byte(got))
}

func TestStatsScreenSwitchViaKey(t *testing.T) {
	t.Parallel()
	m, _ := statsScreenWith(sampleStats())
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 20))
	tm.Type("3")
	tm.Type("q")
	tm.WaitFinished(t)
	got := tm.FinalModel(t).(model).View()
	for _, want := range []string{"Completion", "throughput", "3:stats"} {
		if !strings.Contains(got, want) {
			t.Fatalf("switching to stats should render %q; got:\n%s", want, got)
		}
	}
}

func TestStatsCacheKeyedByResultPointer(t *testing.T) {
	t.Parallel()
	m, ss := statsScreenWith(sampleStats())
	first := strings.Join(ss.contentLines(m.theme, false), "\n")

	ss.res.Scope.Sprint = "mutated-after-first-render"
	again := strings.Join(ss.contentLines(m.theme, false), "\n")
	if again != first {
		t.Fatal("same result pointer must serve the cached lines, ignoring an in-place mutation of the same result")
	}

	other := sampleStats()
	other.Scope.Sprint = "a-distinctly-different-sprint"
	ss.res, ss.loaded = other, true
	fresh := strings.Join(ss.contentLines(m.theme, false), "\n")
	if fresh == first {
		t.Fatal("a fresh result pointer must invalidate the cache and rebuild, not reuse the first render")
	}
	if !strings.Contains(fresh, other.Scope.Sprint) {
		t.Fatalf("rebuilt content must reflect the new result's data, got:\n%s", fresh)
	}
}

func TestLoadStatsFallsBackOnlyOnNoActiveSprint(t *testing.T) {
	t.Parallel()
	s, cfg, _ := initRepo(t)
	createTicket(t, s, cfg, "counted")

	_, err := s.Stats(cfg, core.StatsOpts{Sprint: "active"})
	if !errors.Is(err, core.ErrNoActiveSprint) {
		t.Fatalf("stats without an active sprint must yield ErrNoActiveSprint, got %v", err)
	}
	_, err = s.Stats(cfg, core.StatsOpts{Sprint: "no-such-sprint"})
	if err == nil || errors.Is(err, core.ErrNoActiveSprint) {
		t.Fatalf("an unknown sprint key must not classify as ErrNoActiveSprint, got %v", err)
	}

	res, err := loadStats(s, cfg)
	if err != nil {
		t.Fatalf("loadStats must fall back to unscoped stats when no sprint is active: %v", err)
	}
	if res == nil || res.Completion == nil || res.Completion.Total != 1 {
		t.Fatalf("fallback stats must cover the whole repo, got %+v", res)
	}
}

func TestStatsInvalidateReloads(t *testing.T) {
	t.Parallel()
	_, ss := statsScreenWith(sampleStats())
	if !ss.loaded {
		t.Fatal("statsScreenWith should mark loaded")
	}
	ss.invalidate()
	if ss.loaded {
		t.Fatal("invalidate should clear the loaded flag so the next view reloads")
	}
}
