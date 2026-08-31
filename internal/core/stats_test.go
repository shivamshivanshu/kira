package core

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestResolveScopeSinceUsesRecordedCalendarDay(t *testing.T) {
	s, cfg, _ := stagedFixture(t)
	// Both instants sit within a timezone offset of the since-date boundary:
	// on-day is 2026-07-14T20:30Z and off-day is 2026-07-15T04:00Z, so no
	// single host zone agrees with both recorded days — any host-local
	// interpretation misfilters at least one of them.
	onDay := &datamodel.Item{ID: "IN", Created: "2026-07-15T02:00:00+05:30"}
	offDay := &datamodel.Item{ID: "OUT", Created: "2026-07-14T20:00:00-08:00"}

	_, set, err := s.resolveScope(cfg, StatsOpts{Since: "2026-07-15"}, []*datamodel.Item{onDay, offDay}, nil)
	if err != nil {
		t.Fatalf("resolveScope: %v", err)
	}
	if len(set) != 1 || set[0].ID != "IN" {
		t.Fatalf("--since must filter by each item's recorded calendar day, got %d items %+v", len(set), set)
	}
}

func TestResolveScopeSinceRejectsMalformedDate(t *testing.T) {
	s, cfg, _ := stagedFixture(t)
	if _, _, err := s.resolveScope(cfg, StatsOpts{Since: "not-a-date"}, nil, nil); err == nil {
		t.Fatal("resolveScope must reject a malformed --since value")
	}
}
