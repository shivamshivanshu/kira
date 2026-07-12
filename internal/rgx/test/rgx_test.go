package rgx_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/rgx"
)

func TestParseLine(t *testing.T) {
	cases := []struct {
		name string
		line string
		want rgx.Line
		ok   bool
	}{
		{"match line", ".kira/tickets/01J8.md:14:the snapshot merge", rgx.Line{Path: ".kira/tickets/01J8.md", IsMatch: true, LineNo: 14, Text: "the snapshot merge"}, true},
		{"context line", ".kira/tickets/01J8.md-13-context line", rgx.Line{Path: ".kira/tickets/01J8.md", IsMatch: false, LineNo: 13, Text: "context line"}, true},
		{"separator", "--", rgx.Line{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := rgx.ParseLine(c.line)
			if ok != c.ok {
				t.Fatalf("%q: ok=%v, want %v", c.line, ok, c.ok)
			}
			if got != c.want {
				t.Fatalf("%q: parsed %+v, want %+v", c.line, got, c.want)
			}
		})
	}
}
