package webui

import (
	"net/url"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/seed"
	"github.com/shivamshivanshu/kira/internal/testutil"
)

func BenchmarkLoadPage(b *testing.B) {
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
	dashboard := &server{store: store}
	for _, view := range []string{"tree", "board", "stats", "manage"} {
		b.Run(view, func(b *testing.B) {
			u := &url.URL{Path: "/", RawQuery: url.Values{"view": {view}}.Encode()}
			if _, err := dashboard.loadPage(b.Context(), u); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := dashboard.loadPage(b.Context(), u); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMarkdown(b *testing.B) {
	const source = "# Item description\n\nA **useful** update with a [link](https://example.com).\n\n- [x] Implement\n- [ ] Review\n\n| Owner | State |\n| --- | --- |\n| Ada | Doing |\n"
	if _, err := markdown(source); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := markdown(source); err != nil {
			b.Fatal(err)
		}
	}
}
