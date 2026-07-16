package timex

import (
	"testing"
	"time"
)

func TestOverdue(t *testing.T) {
	now, err := time.Parse(time.DateOnly, "2026-07-16")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		due  string
		want bool
	}{
		{"past date is overdue", "2026-07-15", true},
		{"today is not overdue", "2026-07-16", false},
		{"future date is not overdue", "2026-07-17", false},
		{"empty due is not overdue", "", false},
		{"malformed due is not overdue", "not-a-date", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Overdue(c.due, now); got != c.want {
				t.Errorf("Overdue(%q, %s) = %v, want %v", c.due, now, got, c.want)
			}
		})
	}
}

func TestCompareRFC3339(t *testing.T) {
	t.Run("same instant across offsets", func(t *testing.T) {
		cmp, a, b := CompareRFC3339("2026-01-01T10:00:00+05:30", "2026-01-01T04:30:00Z")
		if !a || !b || cmp != 0 {
			t.Fatalf("identical instants must compare equal: cmp=%d aOK=%v bOK=%v", cmp, a, b)
		}
	})
	t.Run("offset changes ordering", func(t *testing.T) {
		// 09:00+05:30 is 03:30Z, earlier than 05:00Z, though its raw string sorts later
		cmp, _, _ := CompareRFC3339("2026-01-01T09:00:00+05:30", "2026-01-01T05:00:00Z")
		if cmp >= 0 {
			t.Fatalf("03:30Z must sort before 05:00Z, got cmp=%d", cmp)
		}
	})
	t.Run("unparseable reported", func(t *testing.T) {
		cmp, a, b := CompareRFC3339("nonsense", "2026-01-01T05:00:00Z")
		if a || !b || cmp != 0 {
			t.Fatalf("expected only the second to parse: cmp=%d aOK=%v bOK=%v", cmp, a, b)
		}
	})
}
