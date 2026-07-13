package perf

import (
	"fmt"
	"testing"
	"time"
)

const (
	coldStartCeiling = 50 * time.Millisecond
	coldStartSamples = 11
)

func TestColdStartSmoke(t *testing.T) {
	requirePerf(t)
	bin := kiraBin(t)
	dir := fixture(t, 100)

	var best time.Duration
	for i := 0; i < coldStartSamples; i++ {
		d, out, err := runKira(bin, dir, nil, "version")
		if err != nil {
			t.Fatalf("version: %v\n%s", err, out)
		}
		if i == 0 || d < best {
			best = d
		}
	}
	fmt.Printf("\n=== cold-start smoke (min of 11 `kira version`) ===\n  %v (ceiling %v)\n", best, coldStartCeiling)
	if best > coldStartCeiling {
		t.Fatalf("cold start %v exceeds %v order-of-magnitude tripwire", best, coldStartCeiling)
	}
}
