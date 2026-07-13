package perf

import (
	"fmt"
	"strings"
	"testing"
)

func TestSpawnCounts(t *testing.T) {
	requirePerf(t)
	bin := kiraBin(t)
	dir := fixture(t, 1000)
	shimDir, counter := gitShim(t)

	var b strings.Builder
	fmt.Fprintf(&b, "\n=== git-spawn count per command (1k fixture) ===\n")
	for _, c := range commands {
		first := spawnCount(t, bin, dir, shimDir, counter, c.args...)
		second := spawnCount(t, bin, dir, shimDir, counter, c.args...)
		if first != second {
			t.Errorf("%s: spawn count not deterministic (%d then %d)", c.name, first, second)
		}
		fmt.Fprintf(&b, "  %-8s %d\n", c.name, first)
	}
	fmt.Print(b.String())
}
