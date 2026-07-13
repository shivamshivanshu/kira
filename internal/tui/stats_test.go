package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func sampleStats() *datamodel.StatsResult {
	return &datamodel.StatsResult{
		Burndown: &datamodel.Burndown{
			Sprint: "S-1", Start: "2026-07-06", End: "2026-07-17", Unit: "points",
			Days: []datamodel.BurndownDay{
				{Date: "2026-07-06", Remaining: 20, Ideal: 20},
				{Date: "2026-07-07", Remaining: 18, Ideal: 18},
				{Date: "2026-07-08", Remaining: 15, Ideal: 16},
				{Date: "2026-07-09", Remaining: 15, Ideal: 14},
				{Date: "2026-07-10", Remaining: 9, Ideal: 12},
			},
			Unestimated: 2,
		},
		Velocity: &datamodel.Velocity{
			Unit: "points",
			Sprints: []datamodel.VelocitySprint{
				{Key: "S-prev-2", Completed: 12},
				{Key: "S-prev-1", Completed: 18},
				{Key: "S-prev-0", Completed: 9},
			},
			Trailing3: 13,
		},
	}
}

func statsScreenWith(res *datamodel.StatsResult) (model, *statsScreen) {
	m := newTestModel(100, 20, false)
	ss := m.screens[viewStats].(*statsScreen)
	if res != nil {
		ss.setResult(res)
	}
	return m, ss
}

func TestStatsScreenRender(t *testing.T) {
	m, ss := statsScreenWith(sampleStats())
	golden.RequireEqual(t, []byte(ss.view(&m, 100, 18)))
}

func TestStatsScreenEmptyState(t *testing.T) {
	m, ss := statsScreenWith(nil)
	got := ss.view(&m, 100, 18)
	if !strings.Contains(got, "No sprint data") {
		t.Fatalf("empty stats should teach; got:\n%s", got)
	}
	golden.RequireEqual(t, []byte(got))
}

func TestStatsScreenSwitchViaKey(t *testing.T) {
	m, _ := statsScreenWith(sampleStats())
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 20))
	tm.Type("3")
	tm.Type("q")
	tm.WaitFinished(t)
	got := tm.FinalModel(t).(model).View()
	for _, want := range []string{"Burndown", "Velocity", "3:stats"} {
		if !strings.Contains(got, want) {
			t.Fatalf("switching to stats should render %q; got:\n%s", want, got)
		}
	}
}

func TestStatsCacheKeyedByResultPointer(t *testing.T) {
	m, ss := statsScreenWith(sampleStats())
	first := ss.contentLines(m.theme, false)
	again := ss.contentLines(m.theme, false)
	if &first[0] != &again[0] {
		t.Fatal("unchanged result pointer must serve cached lines, not rebuild")
	}
	ss.setResult(sampleStats())
	fresh := ss.contentLines(m.theme, false)
	if &first[0] == &fresh[0] {
		t.Fatal("a fresh result pointer must invalidate the cache; a reload that mutated in place would serve stale lines")
	}
}

func TestStatsInvalidateReloads(t *testing.T) {
	_, ss := statsScreenWith(sampleStats())
	if !ss.loaded {
		t.Fatal("setResult should mark loaded")
	}
	ss.invalidate()
	if ss.loaded {
		t.Fatal("invalidate should clear the loaded flag so the next view reloads")
	}
}
