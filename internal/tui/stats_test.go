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
	if !strings.Contains(got, "No metrics yet") {
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
	for _, want := range []string{"Completion", "throughput", "3:stats"} {
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
