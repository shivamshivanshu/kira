package query

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/item"
)

func orderNums(t *testing.T, expr string, items []*item.Item, cfg *config.Config) []string {
	t.Helper()
	q, err := Parse(expr)
	if err != nil {
		t.Fatalf("Parse(%q): %v", expr, err)
	}
	if q.Order == nil {
		t.Fatalf("Parse(%q): no order clause", expr)
	}
	keyOf := q.Order.Keyer(cfg)
	sorted := make([]*item.Item, len(items))
	copy(sorted, items)
	keys := make(map[*item.Item]OrderKey, len(items))
	for _, it := range sorted {
		keys[it] = keyOf(it)
	}
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && q.Order.Less(keys[sorted[j]], keys[sorted[j-1]]); j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	nums := make([]string, len(sorted))
	for i, it := range sorted {
		nums[i] = it.Number
	}
	return nums
}

// Fixture input order: KIRA-100 (epic: no rank/priority/due/estimate/owner),
// KIRA-1 (P1, rank aam, due 07-20, est 3), KIRA-2 (P0, due 07-01, est 5),
// KIRA-3 (owner only). Pins per-field key semantics and null-last in both
// directions; ties keep input order (stable insertion sort).
func TestOrderBy(t *testing.T) {
	items, _, cfg := fixture()
	tests := []struct {
		expr string
		want string
	}{
		{"x ORDER BY priority", "KIRA-2,KIRA-1,KIRA-100,KIRA-3"},
		{"x ORDER BY priority desc", "KIRA-1,KIRA-2,KIRA-100,KIRA-3"},
		{"x ORDER BY rank", "KIRA-1,KIRA-100,KIRA-2,KIRA-3"},
		{"x ORDER BY rank desc", "KIRA-1,KIRA-100,KIRA-2,KIRA-3"},
		{"x ORDER BY due", "KIRA-2,KIRA-1,KIRA-100,KIRA-3"},
		{"x ORDER BY due desc", "KIRA-1,KIRA-2,KIRA-100,KIRA-3"},
		{"x ORDER BY estimate", "KIRA-1,KIRA-2,KIRA-100,KIRA-3"},
		{"x ORDER BY estimate desc", "KIRA-2,KIRA-1,KIRA-100,KIRA-3"},
		{"x ORDER BY created", "KIRA-3,KIRA-100,KIRA-1,KIRA-2"},
		{"x ORDER BY created desc", "KIRA-2,KIRA-1,KIRA-100,KIRA-3"},
		{"x ORDER BY owner", "KIRA-2,KIRA-1,KIRA-3,KIRA-100"},
	}
	for _, tc := range tests {
		got := orderNums(t, tc.expr, items, cfg)
		if s := strings.Join(got, ","); s != tc.want {
			t.Errorf("%q ordered %s, want %s", tc.expr, s, tc.want)
		}
	}
}
