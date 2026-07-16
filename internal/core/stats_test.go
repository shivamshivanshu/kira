package core

import (
	"testing"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestResolveScopeSinceCrossesTimezoneOffsets(t *testing.T) {
	saved := time.Local
	time.Local = time.FixedZone("ist", 5*60*60+30*60) // +05:30
	defer func() { time.Local = saved }()

	s, cfg, _ := stagedFixture(t)
	// 2026-07-14T20:00:00-08:00 is 2026-07-15T09:30:00+05:30 — after local
	// (IST) midnight on the since-date, even though its own recorded offset
	// makes the raw string start with "2026-07-14".
	edge := &datamodel.Item{ID: "E", Created: "2026-07-14T20:00:00-08:00"}

	_, set, err := s.resolveScope(cfg, StatsOpts{Since: "2026-07-15"}, []*datamodel.Item{edge}, nil)
	if err != nil {
		t.Fatalf("resolveScope: %v", err)
	}
	if len(set) != 1 {
		t.Fatalf("item created after local midnight on the since-date must be included despite its own negative offset, got %d items", len(set))
	}
}

func TestResolveScopeSinceRejectsMalformedDate(t *testing.T) {
	s, cfg, _ := stagedFixture(t)
	if _, _, err := s.resolveScope(cfg, StatsOpts{Since: "not-a-date"}, nil, nil); err == nil {
		t.Fatal("resolveScope must reject a malformed --since value")
	}
}
