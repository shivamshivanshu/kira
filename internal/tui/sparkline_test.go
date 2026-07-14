package tui

import "testing"

func TestSparklineMapsToRamp(t *testing.T) {
	t.Parallel()
	got := sparkline([]float64{0, 1, 2, 3, 4, 5, 6, 7}, true)
	want := "▁▂▃▄▅▆▇█"
	if got != want {
		t.Fatalf("nerd sparkline = %q, want %q", got, want)
	}
	if a := sparkline([]float64{0, 7}, false); a != ".@" {
		t.Fatalf("ascii endpoints = %q, want %q", a, ".@")
	}
}

func TestSparklineFlatAndEmpty(t *testing.T) {
	t.Parallel()
	if s := sparkline(nil, true); s != "" {
		t.Fatalf("empty series = %q, want empty", s)
	}
	if s := sparkline([]float64{0, 0, 0}, true); s != "▁▁▁" {
		t.Fatalf("all-zero series = %q, want lowest level", s)
	}
}
