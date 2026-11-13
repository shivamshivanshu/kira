package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
)

func TestParseFieldFilters(t *testing.T) {
	got, err := parseFieldFilters([]string{"state=TODO", "team=infra=x"})
	if err != nil {
		t.Fatalf("parseFieldFilters: %v", err)
	}
	want := []core.FieldFilter{{Key: "state", Value: "TODO"}, {Key: "team", Value: "infra=x"}}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	for _, bad := range []string{"state", "=TODO", "state="} {
		if _, err := parseFieldFilters([]string{bad}); err == nil {
			t.Errorf("parseFieldFilters(%q) succeeded, want a rejection", bad)
		}
	}
}

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
