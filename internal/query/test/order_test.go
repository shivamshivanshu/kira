package query_test

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/query"
)

func strp(s string) *string   { return &s }
func f64p(f float64) *float64 { return &f }

func fixture() (items []*datamodel.Item, cfg *datamodel.Config) {
	cfg = config.Default()
	epicID := id.Mint().String()
	it1ID := id.Mint().String()
	it2ID := id.Mint().String()
	it3ID := id.Mint().String()

	epic := &datamodel.Item{
		ID: epicID, Number: "KIRA-100", Type: datamodel.TypeEpic, Title: "Big epic",
		State: "ACTIVE", Created: "2026-07-01T00:00:00Z", Updated: "2026-07-01T00:00:00Z",
	}
	it1 := &datamodel.Item{
		ID: it1ID, Number: "KIRA-1", Type: datamodel.TypeTicket, Title: "Fix race in snapshot",
		State: "IN_PROGRESS", Owner: strp("shivam"), Priority: strp("P1"),
		Labels: []string{"bug"}, Epic: strp(epicID),
		Subtype: strp("bug"), Rank: strp("aam"), Sprint: strp("2026-S14"),
		Due: strp("2026-07-20"), Estimate: f64p(3),
		BlockedBy: []string{it2ID}, Links: map[string][]string{string(datamodel.LinkRelates): {it3ID}},
		Created: "2026-07-05T00:00:00Z", Updated: "2026-07-06T00:00:00Z",
	}
	it2 := &datamodel.Item{
		ID: it2ID, Number: "KIRA-2", Type: datamodel.TypeTicket, Title: "Perf tuning",
		State: "TODO", Owner: strp("alice"), Labels: []string{"perf"},
		Priority: strp("P0"), Due: strp("2026-07-01"), Estimate: f64p(5),
		Reporter: strp("shivam"),
		Created:  "2026-07-10T00:00:00Z", Updated: "2026-07-10T00:00:00Z",
	}
	it3 := &datamodel.Item{
		ID: it3ID, Number: "KIRA-3", Type: datamodel.TypeTicket, Title: "Done thing",
		State: "DONE", Owner: strp("shivam"), Labels: []string{"bug", "perf"},
		Resolution: strp("done"),
		Created:    "2026-06-01T00:00:00Z", Updated: "2026-06-02T00:00:00Z",
	}
	return []*datamodel.Item{epic, it1, it2, it3}, cfg
}

func orderNums(t *testing.T, expr string, items []*datamodel.Item, cfg *datamodel.Config) []string {
	t.Helper()
	q, err := query.Parse(expr)
	if err != nil {
		t.Fatalf("Parse(%q): %v", expr, err)
	}
	if q.Order == nil {
		t.Fatalf("Parse(%q): no order clause", expr)
	}
	keyOf := q.Order.Keyer(cfg)
	sorted := make([]*datamodel.Item, len(items))
	copy(sorted, items)
	keys := make(map[*datamodel.Item]query.OrderKey, len(items))
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

func TestOrderBy(t *testing.T) {
	t.Parallel()
	items, cfg := fixture()
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

func TestOrderByBoard(t *testing.T) {
	t.Parallel()
	items, cfg := fixture()
	other := &datamodel.Item{
		ID: id.Mint().String(), Number: "AAA-1", Type: datamodel.TypeTicket, Title: "Other board",
		State: "TODO", Created: "2026-07-01T00:00:00Z", Updated: "2026-07-01T00:00:00Z",
	}
	items = append(items, other)
	tests := []struct {
		expr string
		want string
	}{
		{"x ORDER BY board", "AAA-1,KIRA-100,KIRA-1,KIRA-2,KIRA-3"},
		{"x ORDER BY board desc", "KIRA-100,KIRA-1,KIRA-2,KIRA-3,AAA-1"},
	}
	for _, tc := range tests {
		got := orderNums(t, tc.expr, items, cfg)
		if s := strings.Join(got, ","); s != tc.want {
			t.Errorf("%q ordered %s, want %s", tc.expr, s, tc.want)
		}
	}
}

func TestOrderLessEqualStringKeys(t *testing.T) {
	t.Parallel()
	items, cfg := fixture()
	q, err := query.Parse("x ORDER BY owner")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	keyOf := q.Order.Keyer(cfg)
	a, b := keyOf(items[1]), keyOf(items[3])
	if q.Order.Less(a, b) || q.Order.Less(b, a) {
		t.Errorf("equal string keys (both owner shivam) must not compare Less in either direction")
	}
}
