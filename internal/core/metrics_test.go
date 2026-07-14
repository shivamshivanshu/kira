package core

import (
	"reflect"
	"testing"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func at(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return ts
}

func done(t *testing.T, num, created, doing, doneAt string, degraded bool, reopens int) metricItem {
	t.Helper()
	return metricItem{
		number:   num,
		created:  at(t, created+"T10:00:00Z"),
		doingAt:  at(t, doing+"T10:00:00Z"),
		doneAt:   at(t, doneAt+"T10:00:00Z"),
		hasDoing: true,
		hasDone:  true,
		degraded: degraded,
		category: datamodel.CategoryDone,
		reopens:  reopens,
	}
}

func statsFixture(t *testing.T) []metricItem {
	return []metricItem{
		done(t, "KIRA-1", "2025-12-31", "2026-01-01", "2026-01-03", false, 0),
		done(t, "KIRA-2", "2025-12-31", "2026-01-01", "2026-01-05", false, 0),
		done(t, "KIRA-3", "2025-12-31", "2026-01-01", "2026-01-07", false, 1),
		done(t, "KIRA-4", "2025-12-31", "2026-01-01", "2026-01-11", true, 2),
		{
			number: "KIRA-5", category: datamodel.CategoryDone, dropped: true,
			hasDoing: true, hasDone: true,
			doingAt: at(t, "2026-01-01T10:00:00Z"), doneAt: at(t, "2026-01-04T10:00:00Z"),
		},
		{
			number: "KIRA-6", category: datamodel.CategoryDone, hasDone: true,
			created: at(t, "2026-01-02T10:00:00Z"), doneAt: at(t, "2026-01-02T10:00:00Z"),
		},
		{number: "KIRA-7", category: datamodel.CategoryTodo},
		{number: "KIRA-8", category: datamodel.CategoryDoing, hasDoing: true, doingAt: at(t, "2026-01-10T10:00:00Z")},
		{number: "KIRA-9", category: datamodel.CategoryTodo},
		{number: "KIRA-10", category: datamodel.CategoryTodo},
	}
}

func TestComputeCompletionFixture(t *testing.T) {
	got := computeCompletion(statsFixture(t))
	want := &datamodel.Completion{Done: 5, Total: 10, Dropped: 1, Pct: 0.5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("completion = %+v, want %+v", got, want)
	}
}

func TestComputeCycleFixture(t *testing.T) {
	got := computeCycle(statsFixture(t))
	want := &datamodel.Percentiles{P50: 5, P90: 8.8, N: 4, DegradedN: 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("cycle = %+v, want %+v", got, want)
	}
}

func TestComputeLeadFixture(t *testing.T) {
	got := computeLead(statsFixture(t))
	want := &datamodel.Percentiles{P50: 5, P90: 9.4, N: 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lead = %+v, want %+v", got, want)
	}
}

func TestComputeThroughputFixture(t *testing.T) {
	today := at(t, "2026-01-15T10:00:00Z")
	got := computeThroughput(statsFixture(t), 3, today)
	want := []int{0, 4, 1}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("throughput = %v, want %v", got, want)
	}
}

func TestComputeReopensFixture(t *testing.T) {
	got := computeReopens(statsFixture(t))
	want := &datamodel.Reopens{Count: 3, Items: []string{"KIRA-3", "KIRA-4"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reopens = %+v, want %+v", got, want)
	}
}

func TestPercentilesEmpty(t *testing.T) {
	got := percentiles(nil, 0)
	if got.N != 0 || got.P50 != 0 || got.P90 != 0 {
		t.Errorf("empty percentiles = %+v, want zero", got)
	}
}
