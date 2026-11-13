package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestKnownColumnsRejectsTypo(t *testing.T) {
	t.Parallel()
	known, unknown := knownColumns([]string{"number", "severty"})
	if !slices.Equal(known, []string{"number"}) || !slices.Equal(unknown, []string{"severty"}) {
		t.Fatalf("known=%v unknown=%v", known, unknown)
	}
}

func TestWarnUnknownColumnsIsSilentWhenNothingIsUnknown(t *testing.T) {
	t.Parallel()
	var b bytes.Buffer
	warnUnknownColumns(&b, nil)
	if b.Len() != 0 {
		t.Fatalf("wrote %q for an empty slice", b.String())
	}
	warnUnknownColumns(&b, []string{"severty"})
	if !strings.Contains(b.String(), `ui.list.columns: unknown column "severty"; ignored`) {
		t.Fatalf("warning text = %q", b.String())
	}
}
