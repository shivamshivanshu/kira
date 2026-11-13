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

func TestNumberToULIDLiveNumberWinsOverAlias(t *testing.T) {
	idx, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("opening index: %v", err)
	}
	defer func() { _ = idx.Close() }()
	const (
		keeper = "01HAAAAAAAAAAAAAAAAAAAAAAA"
		loser  = "01HBBBBBBBBBBBBBBBBBBBBBBB"
	)
	insert := `INSERT INTO items (id, number, type, title, state, created, updated)
		VALUES (?,?,'ticket','t','TODO','2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`
	for _, row := range [][2]string{{keeper, "KIRA-5"}, {loser, "KIRA-12"}} {
		if _, err := idx.db.Exec(insert, row[0], row[1]); err != nil {
			t.Fatalf("inserting item: %v", err)
		}
	}
	if _, err := idx.db.Exec(`INSERT INTO aliases (item_id, ord, number) VALUES (?,0,'KIRA-5')`, loser); err != nil {
		t.Fatalf("inserting alias: %v", err)
	}
	numbers, err := idx.numberToULID()
	if err != nil {
		t.Fatalf("numberToULID: %v", err)
	}
	if numbers["KIRA-5"] != keeper {
		t.Errorf("KIRA-5 must resolve to its live holder %s, got %s", keeper, numbers["KIRA-5"])
	}
	if numbers["KIRA-12"] != loser {
		t.Errorf("KIRA-12 must resolve to %s, got %s", loser, numbers["KIRA-12"])
	}
}
