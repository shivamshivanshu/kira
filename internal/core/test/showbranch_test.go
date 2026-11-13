package core_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/showfmt"
)

func TestShowBranchFormMatchesWorkonBranch(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	created := mustCreate(t, s, cfg, "Fix snapshot dedup")

	res, err := s.Workon(cfg, created.Number, core.WorkonOpts{NoMove: true})
	if err != nil {
		t.Fatalf("Workon: %v", err)
	}
	got, err := showfmt.Format(showfmt.FormBranch, showfmt.Item{ID: created.ID, Number: created.Number, Title: created.Title}, cfg.Workon)
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if got != res.Branch {
		t.Fatalf("show --format branch = %q, but workon created %q", got, res.Branch)
	}
}
