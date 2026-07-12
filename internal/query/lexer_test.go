package query

import "testing"

func kinds(toks []token) []tokKind {
	ks := make([]tokKind, len(toks))
	for i, t := range toks {
		ks[i] = t.kind
	}
	return ks
}

func TestLexTokens(t *testing.T) {
	tests := []struct {
		in   string
		want []tokKind
	}{
		{"state=IN_PROGRESS", []tokKind{tokWord, tokEq, tokWord, tokEOF}},
		{"a != b", []tokKind{tokWord, tokNe, tokWord, tokEOF}},
		{"created >= 2026-07-01", []tokKind{tokWord, tokGe, tokWord, tokEOF}},
		{"created<2026-07-01", []tokKind{tokWord, tokLt, tokWord, tokEOF}},
		{"a<=b", []tokKind{tokWord, tokLe, tokWord, tokEOF}},
		{"(a OR b)", []tokKind{tokLParen, tokWord, tokWord, tokWord, tokRParen, tokEOF}},
		{`"quoted term"`, []tokKind{tokString, tokEOF}},
		{"KIRA-100", []tokKind{tokWord, tokEOF}},
		{"2026-07-10T09:14:00+05:30", []tokKind{tokWord, tokEOF}},
		{"   ", []tokKind{tokEOF}},
	}
	for _, tc := range tests {
		toks, err := lex(tc.in)
		if err != nil {
			t.Fatalf("lex(%q): %v", tc.in, err)
		}
		got := kinds(toks)
		if len(got) != len(tc.want) {
			t.Fatalf("lex(%q) kinds = %v, want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("lex(%q) kinds = %v, want %v", tc.in, got, tc.want)
			}
		}
	}
}

func TestLexStringEscapes(t *testing.T) {
	toks, err := lex(`"a \"b\" c"`)
	if err != nil {
		t.Fatal(err)
	}
	if toks[0].kind != tokString || toks[0].text != `a "b" c` {
		t.Fatalf("string text = %q", toks[0].text)
	}
}

func TestLexPositions(t *testing.T) {
	toks, _ := lex("state = TODO")
	wantPos := []int{0, 6, 8, 12}
	for i, p := range wantPos {
		if toks[i].pos != p {
			t.Fatalf("token %d pos = %d, want %d", i, toks[i].pos, p)
		}
	}
}

func TestLexErrors(t *testing.T) {
	tests := []struct {
		in  string
		pos int
	}{
		{"a ! b", 2},  // lone '!'
		{`"abc`, 0},   // unterminated string
		{`"a\xb"`, 2}, // invalid escape
	}
	for _, tc := range tests {
		_, err := lex(tc.in)
		qe, ok := err.(*Error)
		if !ok {
			t.Fatalf("lex(%q) err = %v, want *Error", tc.in, err)
		}
		if qe.Pos != tc.pos {
			t.Fatalf("lex(%q) pos = %d, want %d", tc.in, qe.Pos, tc.pos)
		}
	}
}
