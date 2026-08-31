package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/errx"
)

func TestLabelSetRejectsBlankName(t *testing.T) {
	t.Parallel()
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "Blank label")
	before := mustReadItem(t, s, res.ID)

	for _, label := range []string{"", "   "} {
		_, err := s.LabelSet(cfg, "KIRA-1", label, true, false)
		var e *errx.Error
		if !errors.As(err, &e) || e.Code != errx.ExitUser {
			t.Fatalf("label add %q: want an ExitUser refusal, got %v (%T)", label, err, err)
		}
		if !strings.Contains(e.Error(), "cannot be empty") {
			t.Fatalf("label add %q: unexpected error: %v", label, err)
		}
	}
	if got := mustReadItem(t, s, res.ID); got != before {
		t.Fatalf("a blank label reached the item:\n%s", got)
	}
}
