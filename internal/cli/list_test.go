package cli

import (
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
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
