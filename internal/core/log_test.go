package core

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestInterleaveOrdersByInstantAcrossOffsets(t *testing.T) {
	events := []datamodel.Event{
		{Ts: "2026-01-01T09:00:00+05:30", Field: "state", CommitSHA: "early"},
		{Ts: "2026-01-01T05:00:00Z", Field: "state", CommitSHA: "late"},
	}
	entries := interleave(events, nil)
	if len(entries) != 2 || entries[0].SHA != "late" {
		t.Fatalf("expected the later instant (05:00Z) first; a raw-string sort would order +05:30 first: %+v", entries)
	}
}
