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
		ss.res, ss.state = res, statsReady
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
	ss.res, ss.state = other, statsReady
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

func TestStatsActivateDispatchesAsyncLoadOnce(t *testing.T) {
	t.Parallel()

	// Given a stats screen backed by a real store, not yet loaded.
	s, cfg, _ := initRepo(t)
	createTicket(t, s, cfg, "counted")
	m := newModel(s, cfg, asciiTheme(), iconSet{mode: datamodel.IconText}, false)
	ss, ok := m.statsScreen()
	if !ok {
		t.Fatal("model should hold a *statsScreen for viewStats")
	}

	// When activate is called, it dispatches a load and marks the screen
	// pending instead of blocking on store IO.
	cmd := ss.activate(&m)
	if cmd == nil || ss.state != statsPending {
		t.Fatal("activate should dispatch a load and mark pending on first call")
	}
	if got := ss.view(&m, 40, 5); !strings.Contains(got, "loading") {
		t.Fatalf("view should show a loading placeholder while pending; got:\n%s", got)
	}

	// Then a second call before the load resolves is a no-op, and applying
	// the resolved message clears pending and stores the result.
	if again := ss.activate(&m); again != nil {
		t.Fatal("activate should not dispatch a second time while a load is pending")
	}
	msg, ok := cmd().(statsLoadedMsg)
	if !ok {
		t.Fatalf("dispatched command should resolve to statsLoadedMsg, got %T", cmd())
	}
	ss.applyLoaded(msg)
	if ss.state != statsReady || ss.res == nil {
		t.Fatal("applyLoaded should mark ready and store the result")
	}
}

func TestStatsInvalidateReloads(t *testing.T) {
	t.Parallel()
	_, ss := statsScreenWith(sampleStats())
	if ss.state != statsReady {
		t.Fatal("statsScreenWith should mark ready")
	}
	ss.invalidate()
	if ss.state != statsNotLoaded {
		t.Fatal("invalidate should reset state so the next view reloads")
	}
}
