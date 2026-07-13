package timex

import "testing"

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
