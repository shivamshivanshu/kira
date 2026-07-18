package perf

import (
	"fmt"
	"strings"
	"testing"
)

func scalingSizes() []int {
	if testing.Short() {
		return []int{100, 1000}
	}
	return []int{100, 1000, 5000}
}

func TestScaling(t *testing.T) {
	requirePerf(t)
	bin := kiraBinary(t)
	sizes := scalingSizes()
	shimDir, counter := gitShim(t)

	var b strings.Builder
	fmt.Fprintf(&b, "\n=== git-spawn scaling across fixtures %v ===\n", sizes)
	fmt.Fprintf(&b, "  %-8s %s\n", "command", "spawn-count-per-size (growth = max/min)")
	for _, c := range commands {
		counts := make([]int, len(sizes))
		var minC, maxC int
		for i, sz := range sizes {
			dir := fixture(t, sz)
			counts[i] = spawnCount(t, bin, dir, shimDir, counter, c.args...)
			if i == 0 || counts[i] < minC {
				minC = counts[i]
			}
			if i == 0 || counts[i] > maxC {
				maxC = counts[i]
			}
		}
		growth := "n/a"
		if minC > 0 {
			growth = fmt.Sprintf("%.2fx", float64(maxC)/float64(minC))
		}
		fmt.Fprintf(&b, "  %-8s %v  growth=%s\n", c.name, counts, growth)
	}
	b.WriteString("  (non-gating: a good design holds spawn count ~constant as items grow)\n")
	fmt.Print(b.String())
}
