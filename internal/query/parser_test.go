package query

import "testing"

func TestParseCanonical(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"state=IN_PROGRESS", "(state = IN_PROGRESS)"},
		{"owner != alice", "(owner != alice)"},
		{"created>2026-07-01", "(created > 2026-07-01)"},
		{"created >= 2026-07-01", "(created >= 2026-07-01)"},
		{"updated<2026-07-01", "(updated < 2026-07-01)"},
		{"updated<=2026-07-01", "(updated <= 2026-07-01)"},
		{"due<2026-07-20", "(due < 2026-07-20)"},
		{"estimate>=3.5", "(estimate >= 3.5)"},
		{"priority<=P1", "(priority <= P1)"},
		{"rank=aam", "(rank = aam)"},
		{"sprint=active", "(sprint = active)"},
		{"subtype=bug", "(subtype = bug)"},
		{"resolution=done", "(resolution = done)"},
		{"reporter=alice", "(reporter = alice)"},
		{"blocked_by=KIRA-2", "(blocked_by = KIRA-2)"},
		{"links=KIRA-3", "(links = KIRA-3)"},
		{"priority IN (P0,P1)", "(in priority P0 P1)"},
		{"priority IN (P0, P1, P2)", "(in priority P0 P1 P2)"},
		{"owner in (alice)", "(in owner alice)"},
		{`owner IN ("alice smith", bob)`, "(in owner alice smith bob)"},
		{"sprint IN (active, 2026-S13)", "(in sprint active 2026-S13)"},
		{"owner IS EMPTY", "(is-empty owner)"},
		{"blocked_by IS NOT EMPTY", "(is-not-empty blocked_by)"},
		{"links is empty", "(is-empty links)"},
		{"due IS EMPTY", "(is-empty due)"},
		{"category=doing ORDER BY rank", "(category = doing) (order-by rank asc)"},
		{"category=doing ORDER BY rank asc", "(category = doing) (order-by rank asc)"},
		{"a ORDER BY due desc", "(term a) (order-by due desc)"},
		{"a order by priority DESC", "(term a) (order-by priority desc)"},
		{"blocked_by IS NOT EMPTY ORDER BY priority", "(is-not-empty blocked_by) (order-by priority asc)"},
		{"a OR b ORDER BY created", "(or (term a) (term b)) (order-by created asc)"},
		{"race", "(term race)"},
		{`"two words"`, "(term two words)"},
		{"state", "(term state)"},
		{"order", "(term order)"},
		{"state in flux", "(and (and (term state) (term in)) (term flux))"},
		{"NOT owner=alice", "(not (owner = alice))"},
		{"NOT owner IS EMPTY", "(not (is-empty owner))"},
		{"state=TODO AND owner=shivam", "(and (state = TODO) (owner = shivam))"},
		{"label=bug OR label=perf", "(or (label = bug) (label = perf))"},
		{"race priority=P1", "(and (term race) (priority = P1))"},
		{"a OR b c", "(or (term a) (and (term b) (term c)))"},
		{"a b OR c", "(or (and (term a) (term b)) (term c))"},
		{"a OR b OR c", "(or (or (term a) (term b)) (term c))"},
		{"a b c", "(and (and (term a) (term b)) (term c))"},
		{"(a OR b) c", "(and (or (term a) (term b)) (term c))"},
		{"category=doing AND NOT owner=alice", "(and (category = doing) (not (owner = alice)))"},
		{"a and b", "(and (term a) (term b))"},
		{"a or b", "(or (term a) (term b))"},
		{"not a", "(not (term a))"},
	}
	for _, tc := range tests {
		q, err := Parse(tc.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.in, err)
		}
		if got := q.String(); got != tc.want {
			t.Errorf("Parse(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		pos  int
	}{
		{"empty query", "", 0},
		{"missing value", "state=", 6},
		{"ordering op on a non-comparable field", "owner>alice", 5},
		{"rank is equality-only", "rank>aa", 4},
		{"unparseable date", "created>notadate", 8},
		{"unparseable date on due", "due>notadate", 4},
		{"unparseable number", "estimate>abc", 9},
		{"unclosed paren", "(a", 2},
		{"operator with no left operand", "= a", 0},
		{"leading keyword", "AND x", 0},
		{"trailing operator with no right operand", "a OR", 4},
		{"trailing garbage", "a )", 2},
		{"empty IN list", "priority IN ()", 13},
		{"missing comma in IN list", "priority IN (P0 P1)", 16},
		{"unclosed IN list", "priority IN (P0,", 16},
		{"bad date inside IN", "due IN (2026-07-99)", 8},
		{"bad number inside IN", "estimate IN (x)", 13},
		{"IS without EMPTY", "owner IS full", 9},
		{"state is never empty", "state IS EMPTY", 6},
		{"created is never empty", "created IS EMPTY", 8},
		{"activity is never empty", "activity IS EMPTY", 9},
		{"unknown field before comparison", "onwer=shivam", 0},
		{"no expression before ORDER BY", "ORDER BY rank", 0},
		{"ORDER BY in operand position", "NOT ORDER BY rank", 4},
		{"ORDER BY missing field", "a ORDER BY", 10},
		{"ORDER BY unknown field", "a ORDER BY notafield", 11},
		{"ORDER BY list field", "a ORDER BY label", 11},
		{"trailing garbage after ORDER BY", "a ORDER BY rank desc x", 21},
		{"second ORDER BY clause", "a ORDER BY rank ORDER BY due", 16},
	}
	for _, tc := range tests {
		_, err := Parse(tc.in)
		qe, ok := err.(*Error)
		if !ok {
			t.Fatalf("%s: Parse(%q) err = %v, want *Error", tc.name, tc.in, err)
		}
		if qe.Pos != tc.pos {
			t.Errorf("%s: Parse(%q) pos = %d, want %d (%s)", tc.name, tc.in, qe.Pos, tc.pos, qe.Msg)
		}
	}
}

func TestParseOrderedFieldAllOps(t *testing.T) {
	for _, op := range []string{"=", "!=", ">", ">=", "<", "<="} {
		for _, in := range []string{
			"created" + op + "2026-07-01",
			"updated" + op + "2026-07-01",
			"due" + op + "2026-07-01",
			"estimate" + op + "3",
			"priority" + op + "P1",
		} {
			if _, err := Parse(in); err != nil {
				t.Errorf("%s: %v", in, err)
			}
		}
	}
}
