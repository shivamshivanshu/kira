package core

import (
	"reflect"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestBatchedHistoryMatchesPerFile(t *testing.T) {
	s := eventRepo(t)
	a, b := eventTicket(), eventTicket()
	b.ID, b.Number = "01HZZ0TEST0000000000000001", "KIRA-2"
	commitState(t, s, a, "TODO", "2026-01-05")
	commitState(t, s, b, "TODO", "2026-01-05")
	commitState(t, s, a, "IN_PROGRESS", "2026-01-06")
	commitState(t, s, b, "DONE", "2026-01-07")
	check := func() {
		t.Helper()
		got, err := s.deriveEventBatch([]string{a.ID, b.ID})
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range []*datamodel.Item{a, b} {
			want, err := s.deriveEvents(it.ID)
			if err != nil || !reflect.DeepEqual(got[it.ID], want) {
				t.Fatalf("%s events %v != %v (%v)", it.Number, got[it.ID], want, err)
			}
		}
	}
	check()
	if _, err := s.repo().Output("checkout", "-b", "side"); err != nil {
		t.Fatal(err)
	}
	commitState(t, s, a, "REVIEW", "2026-01-08")
	if _, err := s.repo().Output("checkout", "main"); err != nil {
		t.Fatal(err)
	}
	commitState(t, s, b, "TODO", "2026-01-09")
	if _, err := s.repo().Output("merge", "--no-ff", "side", "-m", "merge"); err != nil {
		t.Fatal(err)
	}
	check()
}
