package core_test

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
)

func TestBoardMoveCrossBoardNeverCollidesWithSource(t *testing.T) {
	for _, style := range []datamodel.IDStyle{datamodel.IDStyleSequential, datamodel.IDStyleHash} {
		style := style
		t.Run(string(style), func(t *testing.T) {
			t.Parallel()
			s, cfg := newStore(t)
			cfg.ID.Style = style

			if _, err := s.BoardCreate(cfg, "OPS", "Ops", ""); err != nil {
				t.Fatalf("board create: %v", err)
			}
			cfg, err := s.Config()
			if err != nil {
				t.Fatalf("reload config: %v", err)
			}
			cfg.ID.Style = style

			tk := mustCreate(t, s, cfg, "movable")

			res, err := s.BoardMove(cfg, tk.ID, "OPS")
			if err != nil {
				t.Fatalf("cross-board move: %v", err)
			}
			if res.From == res.To {
				t.Fatalf("cross-board move produced identical number %q; the second already-on-board guard would need to fire", res.To)
			}
			if !strings.EqualFold(id.KeyOf(res.To), "OPS") {
				t.Fatalf("moved number %q is not on board ops", res.To)
			}

			if _, err := s.BoardMove(cfg, tk.ID, "OPS"); err == nil {
				t.Fatal("moving an item already on the target board must be rejected")
			} else if !strings.Contains(err.Error(), "already on board") {
				t.Fatalf("same-board rejection message = %q, want 'already on board'", err.Error())
			}
		})
	}
}
