package webui

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/seed"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func BenchmarkRenderPage(b *testing.B) {
	for _, n := range []int{100, 1000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			dir := testutil.TemporaryGitRepository(b)
			if _, err := core.Init(dir, "WEB", false); err != nil {
				b.Fatal(err)
			}
			store, err := core.Discover(dir)
			if err != nil {
				b.Fatal(err)
			}
			cfg, err := store.Config()
			if err != nil {
				b.Fatal(err)
			}
			if _, err := seed.Run(dir, cfg, seed.Opts{Size: n, Seed: 20260713}); err != nil {
				b.Fatal(err)
			}
			p, err := (&server{store: store}).loadPage(b.Context(), &url.URL{Path: "/", RawQuery: "view=tree"})
			if err != nil {
				b.Fatal(err)
			}
			var body bytes.Buffer
			if err := renderPage(&body, p); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if err := renderPage(io.Discard, p); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(body.Len()), "html-bytes")
		})
	}
}

// BenchmarkProfileWorkload exercises the shared core through reads, writes and
// UI rendering. It is a PGO experiment workload, not a production traffic trace.
func BenchmarkProfileWorkload(b *testing.B) {
	dir := testutil.TemporaryGitRepository(b)
	if _, err := core.Init(dir, "WEB", false); err != nil {
		b.Fatal(err)
	}
	store, err := core.Discover(dir)
	if err != nil {
		b.Fatal(err)
	}
	cfg, err := store.Config()
	if err != nil {
		b.Fatal(err)
	}
	if _, err := seed.Run(dir, cfg, seed.Opts{Size: 1000, Seed: 20260713}); err != nil {
		b.Fatal(err)
	}
	cfg.Commit.Mode = datamodel.CommitManual
	s := &server{store: store}
	iteration := 0
	b.ReportAllocs()
	for b.Loop() {
		for _, view := range []string{"tree", "board", "stats"} {
			p, err := s.loadPage(b.Context(), &url.URL{Path: "/", RawQuery: "view=" + view + "&item=WEB-1"})
			if err != nil {
				b.Fatal(err)
			}
			if err := renderPage(io.Discard, p); err != nil {
				b.Fatal(err)
			}
		}
		if _, err := store.Edit(cfg, "WEB-1", core.EditOpts{Fields: []core.FieldEdit{{Key: "rank", Value: fmt.Sprint(iteration % 2)}}}); err != nil {
			b.Fatal(err)
		}
		iteration++
	}
}
