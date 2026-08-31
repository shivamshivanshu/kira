package perf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestColdStatsSpawnScaling(t *testing.T) {
	requirePerf(t)
	bin := kiraBinary(t)
	shim, counter := gitShim(t)
	var expected int
	for _, size := range []int{100, 1000} {
		dir := freshFixture(t, size)
		if err := os.RemoveAll(filepath.Join(dir, ".kira", ".cache")); err != nil {
			t.Fatal(err)
		}
		count := spawnCount(t, bin, dir, shim, counter, "stats", "--json")
		t.Logf("cold stats: %d items, %d git processes", size, count)
		if expected != 0 && count != expected {
			t.Fatalf("cold stats git processes grew from %d to %d", expected, count)
		}
		expected = count
	}
}
