package gitx

import (
	"slices"
	"testing"
)

func TestParsePorcelainPaths(t *testing.T) {
	out := " M .kira/tickets/01ABC.md\n" +
		"R  old.md -> new.md\n" +
		"?? \"spa ced.md\"\n" +
		"R  \"old name.md\" -> \"new name.md\"\n"
	got := parsePorcelainPaths(out)
	want := []string{
		".kira/tickets/01ABC.md",
		"new.md",
		"spa ced.md",
		"new name.md",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("parsePorcelainPaths:\n got  %q\n want %q", got, want)
	}
}
