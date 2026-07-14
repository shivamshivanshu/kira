package index_test

import (
	"fmt"
	"testing"
)

func scaleULID(n int) string {
	const enc = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	b := make([]byte, 26)
	for i := range b {
		b[i] = '0'
	}
	for i := 25; n > 0 && i >= 0; i-- {
		b[i] = enc[n&31]
		n >>= 5
	}
	return string(b)
}

func TestIncrementalIndexAtScale(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping 1k-item incremental index build in -short mode")
	}
	const n = 1000
	f := newRepo(t)
	for i := 0; i < n; i++ {
		u := scaleULID(i + 1)
		f.writeTicket(t, u, ticket(u, fmt.Sprintf("KIRA-%d", i+1), fmt.Sprintf("title %d", i+1)))
	}
	f.commit(t, "seed 1000")

	idx := open(t, f)
	if res, items := ensure(t, idx, f); res.Items != n || len(items) != n {
		t.Fatalf("initial build items=%d/%d, want %d", res.Items, len(items), n)
	}

	extra := scaleULID(n + 1)
	f.writeTicket(t, extra, ticket(extra, fmt.Sprintf("KIRA-%d", n+1), "one more"))
	f.commit(t, "one more")

	res, items := ensure(t, idx, f)
	if res.Action != "incremental" {
		t.Fatalf("action=%q, want incremental at the 1k baseline", res.Action)
	}
	if res.Items != n+1 || len(items) != n+1 {
		t.Fatalf("after incremental items=%d/%d, want %d", res.Items, len(items), n+1)
	}
}
