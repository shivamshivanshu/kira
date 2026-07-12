package query

import "testing"

// TestParseCanonical pins the AST shape (prefix, fully parenthesized) for each
// grammar production, and thereby precedence and associativity.
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
		// term fall-through
		{"race", "(term race)"},
		{`"two words"`, "(term two words)"},
		// a field name not followed by a comparison is a title term
		{"state", "(term state)"},
		// not
		{"NOT owner=alice", "(not (owner = alice))"},
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
		e, err := Parse(tc.in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.in, err)
		}
		if got := e.String(); got != tc.want {
			t.Errorf("Parse(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

// TestParseErrors checks each error class reports at the right source position.
func TestParseErrors(t *testing.T) {
	tests := []struct {
		in  string
		pos int
	}{
		{"", 0},                 // empty query
		{"state=", 6},           // missing value
		{"owner>alice", 5},      // ordering op on a non-date field
		{"created>notadate", 8}, // unparseable date
		{"(a", 2},               // unclosed paren
		{"= a", 0},              // operator with no left operand
		{"AND x", 0},            // leading keyword
		{"a OR", 4},             // trailing operator, no right operand
		{"a )", 2},              // trailing garbage
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

// TestParseDateFieldAllOps confirms every comparison is legal on a date field.
func TestParseDateFieldAllOps(t *testing.T) {
	for _, op := range []string{"=", "!=", ">", ">=", "<", "<="} {
		if _, err := Parse("created" + op + "2026-07-01"); err != nil {
			t.Errorf("created%s2026-07-01: %v", op, err)
		}
	}
}
