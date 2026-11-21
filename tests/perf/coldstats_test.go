package perf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivamshivanshu/kira/internal/gitx"
)

func TestColdStatsSpawnScaling(t *testing.T) {
	requirePerf(t)
	bin := kiraBinary(t)
	shim, counter := gitShim(t)
	var expected int
	for _, size := range []int{100, 1000} {
		dir := freshFixture(t, size)
		cache := filepath.Join(dir, ".kira", ".cache")
		if err := os.RemoveAll(cache); err != nil {
			t.Fatal(err)
		}
		linear := spawnCount(t, bin, dir, shim, counter, "stats", "--json")
		if expected != 0 && linear != expected {
			t.Fatalf("linear cold stats git processes grew from %d to %d", expected, linear)
		}
		expected = linear

		mergeOneTicketChange(t, bin, dir)
		if err := os.RemoveAll(cache); err != nil {
			t.Fatal(err)
		}
		merged := spawnCount(t, bin, dir, shim, counter, "stats", "--json")
		t.Logf("cold stats: %d items, %d linear and %d merged git processes", size, linear, merged)
		if merged != linear+1 {
			t.Fatalf("merged cold stats used %d git processes, want %d for the single affected ticket", merged, linear+1)
		}
	}
}

func mergeOneTicketChange(t *testing.T, bin, dir string) {
	t.Helper()
	repo := gitx.Repo{Dir: dir}
	git := func(args ...string) {
		t.Helper()
		if _, err := repo.Output(args...); err != nil {
			t.Fatal(err)
		}
	}
	git("checkout", "-b", "perf-side")
	if _, out, err := runKira(bin, dir, nil, "edit", "KIRA-1", "--rank", "side"); err != nil {
		t.Fatalf("side edit: %v\n%s", err, out)
	}
	git("checkout", "main")
	git("merge", "--no-ff", "-m", "merge one ticket change", "perf-side")
}
