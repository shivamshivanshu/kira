package cli

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestRenderWarningExhaustive(t *testing.T) {
	for _, code := range datamodel.WarnCodes {
		w := datamodel.Warning{Code: code, Args: []string{"a", "b"}}
		if renderWarning(w) == "" {
			t.Errorf("renderWarning(%q) empty; add a case", code)
		}
	}
	if renderWarning(datamodel.Warning{Code: "unhandled"}) != "" {
		t.Error("renderWarning(unhandled) non-empty")
	}
}

func TestRenderDiffHeaderExhaustive(t *testing.T) {
	want := map[datamodel.DiffStatus]string{
		datamodel.DiffCreated: "created 7  hi\n",
		datamodel.DiffDeleted: "deleted 7  hi\n",
		datamodel.DiffChanged: "7  hi\n",
	}
	for _, st := range datamodel.DiffStatuses {
		exp, ok := want[st]
		if !ok {
			t.Fatalf("no expected header for %q; add a case", st)
		}
		var b strings.Builder
		renderDiffHeader(&b, st, "7", "hi")
		if b.String() != exp {
			t.Errorf("renderDiffHeader(%q) = %q, want %q", st, b.String(), exp)
		}
	}
}
