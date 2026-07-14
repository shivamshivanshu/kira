package tui

import "testing"

func TestTreeWidthSplit(t *testing.T) {
	cases := []struct {
		width int
		split float64
		want  int
	}{
		{100, 0.5, 50},
		{100, 0.75, 75},
		{100, 0.3, 40},
		{100, 0, 50},
		{100, 1, 50},
		{100, 1.5, 50},
		{79, 0.3, 79},
	}
	for _, c := range cases {
		if got := treeWidth(c.width, c.split); got != c.want {
			t.Errorf("treeWidth(%d, %v) = %d, want %d", c.width, c.split, got, c.want)
		}
	}
}
