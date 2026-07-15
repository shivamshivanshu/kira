package statsfmt_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/statsfmt"
)

func TestCompletionLine(t *testing.T) {
	cases := []struct {
		name string
		c    *datamodel.Completion
		want string
	}{
		{"no dropped", &datamodel.Completion{Done: 3, Total: 4, Pct: 0.75}, "3/4 done (75%)"},
		{"with dropped", &datamodel.Completion{Done: 3, Total: 4, Pct: 0.75, Dropped: 2}, "3/4 done (75%), 2 dropped"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := statsfmt.CompletionLine(tc.c); got != tc.want {
				t.Errorf("CompletionLine() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPercentileLine(t *testing.T) {
	cases := []struct {
		name string
		p    *datamodel.Percentiles
		want string
	}{
		{"integral values", &datamodel.Percentiles{P50: 2, P90: 5, N: 10}, "p50 2  p90 5  n=10"},
		{"fractional values", &datamodel.Percentiles{P50: 1.5, P90: 3.25, N: 7}, "p50 1.5  p90 3.25  n=7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := statsfmt.PercentileLine(tc.p); got != tc.want {
				t.Errorf("PercentileLine() = %q, want %q", got, tc.want)
			}
		})
	}
}
