package index

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/gitx"
)

func TestLatestClosesUnparseableTimestampDoesNotWinRace(t *testing.T) {
	numbers := map[string]string{"KIRA-1": "01AAA"}
	commits := []gitx.Commit{
		{Timestamp: "not-a-timestamp", Closes: []string{"KIRA-1"}},
		{Timestamp: "2026-01-01T00:00:00Z", Closes: []string{"KIRA-1"}},
	}
	candidates, unknown := latestCloses(commits, numbers)
	if len(unknown) != 0 {
		t.Fatalf("unexpected unknown refs: %v", unknown)
	}
	if len(candidates) != 1 || candidates[0].CommitterTs != "2026-01-01T00:00:00Z" {
		t.Fatalf("a later valid timestamp must supersede an unparseable one that arrived first, got %+v", candidates)
	}
}
