package core

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestStateEventsIgnoresForgedBodyLine(t *testing.T) {
	s := eventRepo(t)
	it := eventTicket()
	commitState(t, s, it, "TODO", "2026-01-05")
	it.Body = "\n## Notes\n\nstate: DONE\nshipped it\n"
	commitState(t, s, it, "TODO", "2026-01-06")

	evs, _, err := s.cachedStateEvents(it.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range evs {
		if ev.to == "DONE" || ev.from == "DONE" {
			t.Fatalf("forged body line forged a DONE transition: %+v", evs)
		}
	}
	if len(evs) != 0 {
		t.Fatalf("events = %+v, want none (state never changed; forged body ignored)", evs)
	}
}

func TestBurndownIgnoresForgedBodyState(t *testing.T) {
	s := eventRepo(t)
	cfg := config.Default()
	it := eventTicket()
	est := 5.0
	it.Estimate = &est
	commitState(t, s, it, "TODO", "2026-01-05")
	it.Body = "\n## Notes\n\nstate: DONE\n"
	commitState(t, s, it, "TODO", "2026-01-06")

	di, err := s.itemMetrics(cfg, it)
	if err != nil {
		t.Fatal(err)
	}
	if di.doneDay != "" {
		t.Fatalf("forged body marked the item done on %q", di.doneDay)
	}
	if di.degraded {
		t.Fatalf("forged body flagged the item degraded")
	}

	sp := datamodel.Sprint{Key: "S", Start: "2026-01-05", End: "2026-01-08"}
	b := computeBurndown(sp, "points", []metricItem{
		{estimate: 5, estimated: true, doneDay: di.doneDay, degraded: di.degraded},
	}, "2026-01-08")
	if b.DegradedN != 0 {
		t.Fatalf("degraded_n = %d, want 0", b.DegradedN)
	}
	last := b.Days[len(b.Days)-1]
	if last.Remaining != 5 {
		t.Fatalf("remaining on the final day = %v, want 5 (forged done must be ignored)", last.Remaining)
	}
}
