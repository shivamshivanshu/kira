package tui

import "testing"

func TestSparklineMapsToRamp(t *testing.T) {
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
	if s := sparkline(nil, true); s != "" {
		t.Fatalf("empty series = %q, want empty", s)
	}
	if s := sparkline([]float64{0, 0, 0}, true); s != "▁▁▁" {
		t.Fatalf("all-zero series = %q, want lowest level", s)
	}
}

func TestHbarScales(t *testing.T) {
	if b := hbar(4, 8, 8, true); b != "████    " {
		t.Fatalf("half bar = %q", b)
	}
	if b := hbar(0, 8, 4, false); b != "    " {
		t.Fatalf("zero bar = %q", b)
	}
	if b := hbar(8, 8, 4, false); b != "####" {
		t.Fatalf("full bar = %q", b)
	}
}
