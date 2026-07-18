package perf

import (
	"os"
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

func BenchmarkCommands(b *testing.B) {
	if !gitx.Installed() {
		b.Skip("git not installed")
	}
	bin := kiraBinary(b)
	dir := fixture(b, benchSize())
	for _, c := range commands {
		b.Run(c.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, out, err := runKira(bin, dir, nil, c.args...); err != nil {
					b.Fatalf("%s: %v\n%s", c.name, err, out)
				}
			}
		})
	}
}
