package cli

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestWarningOutputPrefixes(t *testing.T) {
	warnings := []datamodel.Warning{{Code: datamodel.WarnCloseFailed, Args: []string{"KIRA-2", "boom"}}}
	var mutations, notes strings.Builder
	emitMutationWarnings(&mutations, warnings)
	emitStderrNotes(&notes, warnings)
	if got, want := mutations.String(), msgPrefix+" warning: failed to close KIRA-2: boom\n"; got != want {
		t.Errorf("mutation warnings = %q, want %q", got, want)
	}
	if got, want := notes.String(), msgPrefix+" failed to close KIRA-2: boom\n"; got != want {
		t.Errorf("stderr notes = %q, want %q", got, want)
	}
}

func TestRenderSkew(t *testing.T) {
	sk := &datamodel.Skew{Ref: "KIRA-1", At: "HEAD~2", AtID: "01AAA", NowID: "01BBB"}
	want := "KIRA-1 at HEAD~2 is 01AAA; currently it is a different item (01BBB)"
	if got := renderSkew(sk); got != want {
		t.Errorf("renderSkew = %q, want %q", got, want)
	}
}
