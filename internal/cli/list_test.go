package cli

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
)

func TestPinnedColumnsIgnoresTheBuiltInDefault(t *testing.T) {
	cfg := &datamodel.Config{}
	cfg.UI.List.Columns = slices.Clone(datamodel.DefaultListColumns)
	if got := pinnedColumns(cfg); got != nil {
		t.Errorf("pinnedColumns(default) = %v, want nil", got)
	}
	cfg.UI.List.Columns = []string{"number", "severity"}
	if want := []string{"number", "severity"}; !slices.Equal(pinnedColumns(cfg), want) {
		t.Errorf("pinnedColumns(custom) = %v, want %v", pinnedColumns(cfg), want)
	}
}

func TestResolveColumnsPrefersConfiguredThenSchemaThenDefault(t *testing.T) {
	configured := []string{"number", "severity"}
	declared := []string{"number", "watchers", "title"}
	if !slices.Equal(resolveColumns(configured, declared), configured) {
		t.Errorf("configured did not win: %v", resolveColumns(configured, declared))
	}
	if !slices.Equal(resolveColumns(nil, declared), declared) {
		t.Errorf("schema-declared did not win: %v", resolveColumns(nil, declared))
	}
	if !slices.Equal(resolveColumns(nil, nil), datamodel.DefaultListColumns) {
		t.Errorf("fallback = %v, want %v", resolveColumns(nil, nil), datamodel.DefaultListColumns)
	}
}

func TestFormatItemRowRendersSchemaDeclaredFields(t *testing.T) {
	cols := []string{"number", "severity", "watchers", "points"}
	set := datamodel.ListItem{
		Number: "KIRA-1",
		Fields: map[string]any{"severity": "HIGH", "watchers": []string{"alice", "bob"}, "points": 8},
	}
	if got, want := formatItemRow(cols, set), "KIRA-1\tHIGH\talice, bob\t8"; got != want {
		t.Errorf("declared fields: got %q, want %q", got, want)
	}
	absent := datamodel.ListItem{Number: "KIRA-2"}
	if got, want := formatItemRow(cols, absent), "KIRA-2\t-\t-\t-"; got != want {
		t.Errorf("absent fields: got %q, want %q", got, want)
	}
}

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

func columnSchemas() map[string]entityschema.Schema {
	return map[string]entityschema.Schema{
		datamodel.TypeTicket: {Name: datamodel.TypeTicket, Fields: []entityschema.FieldDef{
			{Name: "severity", Type: entityschema.FieldString, Placement: entityschema.PlacementFrontmatter},
			{Name: "notes", Type: entityschema.FieldMarkdown, Placement: entityschema.PlacementBody, Section: "Notes"},
		}},
	}
}

func TestKnownColumnsAcceptsCellsAndDeclaredFields(t *testing.T) {
	t.Parallel()
	known, unknown := knownColumns([]string{"number", "title", "severity"}, columnSchemas())
	if !slices.Equal(known, []string{"number", "title", "severity"}) || len(unknown) != 0 {
		t.Fatalf("known=%v unknown=%v", known, unknown)
	}
}

func TestKnownColumnsRejectsTypoAndBodyPlacedField(t *testing.T) {
	t.Parallel()
	known, unknown := knownColumns([]string{"number", "notes", "severty"}, columnSchemas())
	if !slices.Equal(known, []string{"number"}) {
		t.Fatalf("known = %v, want only the built-in cell", known)
	}
	// notes is declared but body-placed: CustomFields skips it, so itemCell
	// has nothing to render and it is as unusable as the typo.
	if !slices.Equal(unknown, []string{"notes", "severty"}) {
		t.Fatalf("unknown = %v", unknown)
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
