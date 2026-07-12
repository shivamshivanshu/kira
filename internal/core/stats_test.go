package core

import (
	"reflect"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
)

var burnSprint = config.Sprint{Key: "S1", Name: "Sprint 1", Start: "2026-01-05", End: "2026-01-09"}

var burnItems = []burnItem{
	{estimate: 3, estimated: true, doneDay: "2026-01-07"},
	{estimate: 5, estimated: true, doneDay: "2026-01-08"},
	{estimate: 2, estimated: true},
	{estimate: 7, estimated: true, doneDay: "2026-01-06"},
	{estimated: false},
}

func TestComputeBurndownFullWindow(t *testing.T) {
	b := computeBurndown(burnSprint, "points", burnItems, "2026-07-12")
	want := []BurndownDay{
		{Date: "2026-01-05", Remaining: 17, Ideal: 17},
		{Date: "2026-01-06", Remaining: 10, Ideal: 12.8},
		{Date: "2026-01-07", Remaining: 7, Ideal: 8.5},
		{Date: "2026-01-08", Remaining: 2, Ideal: 4.3},
		{Date: "2026-01-09", Remaining: 2, Ideal: 0},
	}
	if !reflect.DeepEqual(b.Days, want) {
		t.Errorf("days = %+v\nwant %+v", b.Days, want)
	}
	if b.Sprint != "S1" || b.Start != "2026-01-05" || b.End != "2026-01-09" || b.Unit != "points" {
		t.Errorf("header = %+v", b)
	}
	if b.Unestimated != 1 || b.DegradedN != 0 {
		t.Errorf("unestimated = %d, degraded_n = %d; want 1, 0", b.Unestimated, b.DegradedN)
	}
}

func TestComputeBurndownTruncatesAtToday(t *testing.T) {
	b := computeBurndown(burnSprint, "points", burnItems, "2026-01-07")
	if len(b.Days) != 3 || b.Days[2].Date != "2026-01-07" || b.Days[2].Remaining != 7 {
		t.Errorf("days = %+v, want series ending at 2026-01-07 remaining 7", b.Days)
	}
}

func TestComputeBurndownBeforeStart(t *testing.T) {
	b := computeBurndown(burnSprint, "points", burnItems, "2026-01-04")
	if len(b.Days) != 0 || b.Days == nil {
		t.Errorf("days = %#v, want empty non-nil series", b.Days)
	}
}

func TestComputeBurndownDoneBeforeStartExcludedFromDayOne(t *testing.T) {
	items := []burnItem{
		{estimate: 4, estimated: true, doneDay: "2026-01-02"},
		{estimate: 6, estimated: true},
	}
	b := computeBurndown(burnSprint, "points", items, "2026-01-05")
	if b.Days[0].Remaining != 6 || b.Days[0].Ideal != 6 {
		t.Errorf("day one = %+v, want remaining 6 ideal 6", b.Days[0])
	}
}

func TestComputeBurndownCountsDegraded(t *testing.T) {
	items := []burnItem{{estimate: 1, estimated: true, doneDay: "2026-01-06", degraded: true}}
	b := computeBurndown(burnSprint, "points", items, "2026-01-06")
	if b.DegradedN != 1 {
		t.Errorf("degraded_n = %d, want 1", b.DegradedN)
	}
}

func TestComputeVelocity(t *testing.T) {
	closed := []config.Sprint{
		{Key: "S1", Start: "2026-01-05", End: "2026-01-09"},
		{Key: "S2", Start: "2026-01-12", End: "2026-01-16"},
	}
	bySprint := map[string][]velocityItem{
		"S1": {
			{estimate: 3, doneDay: "2026-01-07"},
			{estimate: 5, doneDay: "2026-01-08"},
			{estimate: 7, doneDay: "2026-01-06", dropped: true},
			{estimate: 2},
			{estimate: 9, doneDay: "2026-02-01"},
		},
		"S2": {{estimate: 4, doneDay: "2026-01-16"}},
	}
	v := computeVelocity(closed, "points", bySprint)
	want := []VelocitySprint{{Key: "S1", Completed: 8}, {Key: "S2", Completed: 4}}
	if !reflect.DeepEqual(v.Sprints, want) {
		t.Errorf("sprints = %+v, want %+v", v.Sprints, want)
	}
	if v.Trailing3 != 6 {
		t.Errorf("trailing_3 = %v, want 6 (mean of 8, 4)", v.Trailing3)
	}
	if v.Unit != "points" {
		t.Errorf("unit = %q", v.Unit)
	}
}

func TestComputeVelocityTrailingThreeOfFour(t *testing.T) {
	var closed []config.Sprint
	bySprint := map[string][]velocityItem{}
	for i, key := range []string{"A", "B", "C", "D"} {
		sp := config.Sprint{Key: key, Start: "2026-01-05", End: "2026-01-09"}
		closed = append(closed, sp)
		bySprint[key] = []velocityItem{{estimate: float64(10 * (i + 1)), doneDay: "2026-01-06"}}
	}
	v := computeVelocity(closed, "points", bySprint)
	if v.Trailing3 != 30 {
		t.Errorf("trailing_3 = %v, want 30", v.Trailing3)
	}
}

func TestComputeVelocityNoClosedSprints(t *testing.T) {
	v := computeVelocity(nil, "points", nil)
	if v.Sprints == nil || len(v.Sprints) != 0 || v.Trailing3 != 0 {
		t.Errorf("velocity = %+v, want empty non-nil sprints and 0 average", v)
	}
}

func TestComputeVelocityRoundsTrailing(t *testing.T) {
	closed := []config.Sprint{
		{Key: "A", Start: "2026-01-05", End: "2026-01-09"},
		{Key: "B", Start: "2026-01-12", End: "2026-01-16"},
		{Key: "C", Start: "2026-01-19", End: "2026-01-23"},
	}
	bySprint := map[string][]velocityItem{
		"A": {{estimate: 28, doneDay: "2026-01-06"}},
		"B": {{estimate: 31, doneDay: "2026-01-13"}},
		"C": {{estimate: 26, doneDay: "2026-01-20"}},
	}
	if v := computeVelocity(closed, "points", bySprint); v.Trailing3 != 28.3 {
		t.Errorf("trailing_3 = %v, want 28.3", v.Trailing3)
	}
}

func TestClosedSprints(t *testing.T) {
	cfg := config.Default()
	cfg.Sprints = []config.Sprint{
		{Key: "future", Start: "2026-08-01", End: "2026-08-14"},
		{Key: "later", Start: "2026-03-01", End: "2026-03-14"},
		{Key: "earlier", Start: "2026-01-05", End: "2026-01-18"},
		{Key: "ends-today", Start: "2026-07-01", End: "2026-07-12"},
	}
	got := closedSprints(cfg, "2026-07-12")
	if len(got) != 2 || got[0].Key != "earlier" || got[1].Key != "later" {
		t.Errorf("closed = %+v, want [earlier later] (end-ascending, end day passed)", got)
	}
}
