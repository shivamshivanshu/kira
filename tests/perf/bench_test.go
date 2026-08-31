package perf

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/shivamshivanshu/kira/internal/gitx"
)

func benchSize() int {
	if v := os.Getenv("KIRA_BENCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 1000
}

// Fixture copies and cleanup are excluded; every timed operation sees the same
// starting state instead of gradually growing history across iterations.
func BenchmarkMutations(b *testing.B) {
	bin := kiraBinary(b)
	for _, c := range append(append([]command(nil), mutations...), command{"bulk-edit", []string{"edit", "KIRA-1", "KIRA-2", "KIRA-3", "KIRA-4", "KIRA-5", "--field", "rank=perf", "--json"}}) {
		b.Run(c.name, func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				dir := freshFixture(b, benchSize())
				b.StartTimer()
				_, out, err := runKira(bin, dir, nil, c.args...)
				b.StopTimer()
				removeErr := os.RemoveAll(dir)
				if err != nil {
					b.Fatalf("%s: %v\n%s", c.name, err, out)
				}
				if removeErr != nil {
					b.Fatal(removeErr)
				}
				b.StartTimer()
			}
		})
	}
}

func BenchmarkStatsCache(b *testing.B) {
	bin := kiraBinary(b)
	for _, state := range []string{"cold", "warm", "one-change"} {
		b.Run(state, func(b *testing.B) {
			for b.Loop() {
				b.StopTimer()
				dir := freshFixture(b, benchSize())
				switch state {
				case "cold":
					if err := os.RemoveAll(filepath.Join(dir, ".kira", ".cache")); err != nil {
						b.Fatal(err)
					}
				case "one-change":
					if _, out, err := runKira(bin, dir, nil, "edit", "KIRA-1", "--field", "rank=changed"); err != nil {
						b.Fatalf("edit: %v\n%s", err, out)
					}
				}
				b.StartTimer()
				_, out, err := runKira(bin, dir, nil, "stats", "--json")
				b.StopTimer()
				removeErr := os.RemoveAll(dir)
				if err != nil {
					b.Fatalf("stats: %v\n%s", err, out)
				}
				if removeErr != nil {
					b.Fatal(removeErr)
				}
				b.StartTimer()
			}
		})
	}
}

func BenchmarkCommands(b *testing.B) {
	if !gitx.Installed() {
		b.Skip("git not installed")
	}
	bin := kiraBinary(b)
	dir := fixture(b, benchSize())
	for _, c := range commands {
		b.Run(c.name, func(b *testing.B) {
			for b.Loop() {
				if _, out, err := runKira(bin, dir, nil, c.args...); err != nil {
					b.Fatalf("%s: %v\n%s", c.name, err, out)
				}
			}
		})
	}
}
