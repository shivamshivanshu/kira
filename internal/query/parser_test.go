package query

import "testing"

func TestParseCanonical(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		// predicates, one per comparison form
		{"state=IN_PROGRESS", "(state = IN_PROGRESS)"},
		{"owner != alice", "(owner != alice)"},
		{"created>2026-07-01", "(created > 2026-07-01)"},
		{"created >= 2026-07-01", "(created >= 2026-07-01)"},
		{"updated<2026-07-01", "(updated < 2026-07-01)"},
		{"updated<=2026-07-01", "(updated <= 2026-07-01)"},
		// M1.5 fields compare
		{"due<2026-07-20", "(due < 2026-07-20)"},
		{"estimate>=3.5", "(estimate >= 3.5)"},
		{"priority<=P1", "(priority <= P1)"}, // ranked legality checked at compile
		{"rank=aam", "(rank = aam)"},
		{"sprint=active", "(sprint = active)"},
		{"subtype=bug", "(subtype = bug)"},
		{"resolution=done", "(resolution = done)"},
		{"reporter=alice", "(reporter = alice)"},
		{"blocked_by=KIRA-2", "(blocked_by = KIRA-2)"},
		{"links=KIRA-3", "(links = KIRA-3)"},
		// IN membership
		{"priority IN (P0,P1)", "(in priority P0 P1)"},
		{"priority IN (P0, P1, P2)", "(in priority P0 P1 P2)"},
		{"owner in (alice)", "(in owner alice)"},
		{`owner IN ("alice smith", bob)`, "(in owner alice smith bob)"},
		{"sprint IN (active, 2026-S13)", "(in sprint active 2026-S13)"},
		// IS [NOT] EMPTY
		{"owner IS EMPTY", "(is-empty owner)"},
		{"blocked_by IS NOT EMPTY", "(is-not-empty blocked_by)"},
		{"links is empty", "(is-empty links)"},
		{"due IS EMPTY", "(is-empty due)"},
		// ORDER BY: trailing, optional direction, case-insensitive
		{"category=doing ORDER BY rank", "(category = doing) (order-by rank asc)"},
		{"category=doing ORDER BY rank asc", "(category = doing) (order-by rank asc)"},
		{"a ORDER BY due desc", "(term a) (order-by due desc)"},
		{"a order by priority DESC", "(term a) (order-by priority desc)"},
		{"blocked_by IS NOT EMPTY ORDER BY priority", "(is-not-empty blocked_by) (order-by priority asc)"},
		{"a OR b ORDER BY created", "(or (term a) (term b)) (order-by created asc)"},
		// term fall-through
		{"race", "(term race)"},
		{`"two words"`, "(term two words)"},
		// a field name not followed by a comparison is a title term
		{"state", "(term state)"},
		// a lone `order` (no BY) is a title term, as is `in` without '('
		{"order", "(term order)"},
		{"state in flux", "(and (and (term state) (term in)) (term flux))"},
		// not
		{"NOT owner=alice", "(not (owner = alice))"},
		{"NOT owner IS EMPTY", "(not (is-empty owner))"},
		// explicit and, or
		{"state=TODO AND owner=shivam", "(and (state = TODO) (owner = shivam))"},
		{"label=bug OR label=perf", "(or (label = bug) (label = perf))"},
		// adjacency implies AND
		{"race priority=P1", "(and (term race) (priority = P1))"},
		// precedence: AND binds tighter than OR
		{"a OR b c", "(or (term a) (and (term b) (term c)))"},
		{"a b OR c", "(or (and (term a) (term b)) (term c))"},
		// left associativity
		{"a OR b OR c", "(or (or (term a) (term b)) (term c))"},
		{"a b c", "(and (and (term a) (term b)) (term c))"},
		// parentheses override precedence
		{"(a OR b) c", "(and (or (term a) (term b)) (term c))"},
		{"category=doing AND NOT owner=alice", "(and (category = doing) (not (owner = alice)))"},
		// keywords are case-insensitive
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
		in  string
		pos int
	}{
		{"", 0},                 // empty query
		{"state=", 6},           // missing value
		{"owner>alice", 5},      // ordering op on a non-comparable field
		{"rank>aa", 4},          // rank is equality-only
		{"created>notadate", 8}, // unparseable date
		{"due>notadate", 4},     // unparseable date on due
		{"estimate>abc", 9},     // unparseable number
		{"(a", 2},               // unclosed paren
		{"= a", 0},              // operator with no left operand
		{"AND x", 0},            // leading keyword
		{"a OR", 4},             // trailing operator, no right operand
		{"a )", 2},              // trailing garbage
		// IN
		{"priority IN ()", 13},      // empty value list
		{"priority IN (P0 P1)", 16}, // missing comma
		{"priority IN (P0,", 16},    // unclosed list
		{"due IN (2026-07-99)", 8},  // bad date inside IN
		{"estimate IN (x)", 13},     // bad number inside IN
		// IS EMPTY
		{"owner IS full", 9},    // IS without EMPTY
		{"state IS EMPTY", 6},   // state is never empty
		{"created IS EMPTY", 8}, // created is never empty
		// ORDER BY
		{"ORDER BY rank", 0},                 // no expression before the clause
		{"NOT ORDER BY rank", 4},             // operand position
		{"a ORDER BY", 10},                   // missing field
		{"a ORDER BY notafield", 11},         // unknown field
		{"a ORDER BY label", 11},             // list field
		{"a ORDER BY rank desc x", 21},       // trailing garbage after the clause
		{"a ORDER BY rank ORDER BY due", 16}, // only one clause, trailing
	}
	for _, tc := range tests {
		_, err := Parse(tc.in)
		qe, ok := err.(*Error)
		if !ok {
			t.Fatalf("Parse(%q) err = %v, want *Error", tc.in, err)
		}
		if qe.Pos != tc.pos {
			t.Errorf("Parse(%q) pos = %d, want %d (%s)", tc.in, qe.Pos, tc.pos, qe.Msg)
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
