package errx

import "testing"

func TestNearest(t *testing.T) {
	states := []string{"TODO", "IN_PROGRESS", "REVIEW", "DONE", "WONT_DO"}
	cases := []struct {
		name       string
		input      string
		candidates []string
		want       string
	}{
		{"exact", "DONE", states, "DONE"},
		{"one-edit", "DONW", states, "DONE"},
		{"two-edits", "REVIEWED", states, "REVIEW"},
		{"beyond-threshold", "INPROG", states, ""},
		{"no-candidates", "DONE", nil, ""},
		{"nothing-close", "zzzzz", states, ""},
		{"number-typo", "KIRA-1X", []string{"KIRA-1", "KIRA-2"}, "KIRA-1"},
		{"number-far", "KIRA-999", []string{"KIRA-1", "KIRA-2", "KIRA-3"}, ""},
		{"closest-of-many", "REVEW", states, "REVIEW"},
		{"wrong-case", "done", states, "DONE"},
		{"wrong-case-typo", "don", states, "DONE"},
		{"threshold-scales-by-runes-not-bytes", "日本語", []string{"日曜日"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Nearest(c.input, c.candidates); got != c.want {
				t.Errorf("Nearest(%q, %v) = %q, want %q", c.input, c.candidates, got, c.want)
			}
		})
	}
}

func TestEditDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "ab", 1},
		{"kitten", "sitting", 3},
		{"", "abc", 3},
	}
	for _, c := range cases {
		if got := editDistance([]rune(c.a), []rune(c.b)); got != c.want {
			t.Errorf("editDistance(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
