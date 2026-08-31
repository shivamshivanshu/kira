package cli

import (
	"bytes"
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
)

func extrasSchemas() map[string]entityschema.Schema {
	return map[string]entityschema.Schema{
		"ticket": {Name: "ticket", Fields: []entityschema.FieldDef{
			{Name: "severity", Type: entityschema.FieldString, Placement: entityschema.PlacementFrontmatter},
			{Name: "watchers", Type: entityschema.FieldString, List: true, Placement: entityschema.PlacementFrontmatter},
			{Name: "description", Type: entityschema.FieldMarkdown, Placement: entityschema.PlacementBody},
		}},
	}
}

func TestResolveColumnsAcceptsDeclaredExtras(t *testing.T) {
	cols, unknown := resolveColumns([]string{"number", "severity", "watchers", "title"}, extrasSchemas())
	if want := []string{"number", "severity", "watchers", "title"}; !slices.Equal(cols, want) {
		t.Errorf("cols = %v, want %v", cols, want)
	}
	if len(unknown) != 0 {
		t.Errorf("unknown = %v, want none", unknown)
	}
}

func TestResolveColumnsRejectsUnknownAndBodyFields(t *testing.T) {
	cols, unknown := resolveColumns([]string{"number", "severty", "description"}, extrasSchemas())
	if want := []string{"number"}; !slices.Equal(cols, want) {
		t.Errorf("cols = %v, want %v", cols, want)
	}
	if want := []string{"severty", "description"}; !slices.Equal(unknown, want) {
		t.Errorf("unknown = %v, want %v", unknown, want)
	}
}

func TestResolveColumnsDefaultsWhenNothingResolves(t *testing.T) {
	cols, _ := resolveColumns([]string{"bogus"}, nil)
	if !slices.Equal(cols, datamodel.DefaultListColumns) {
		t.Errorf("cols = %v, want defaults %v", cols, datamodel.DefaultListColumns)
	}
}

func TestResolveColumnsKeepsDefaultsByteIdentical(t *testing.T) {
	cols, unknown := resolveColumns(datamodel.DefaultListColumns, extrasSchemas())
	if !slices.Equal(cols, datamodel.DefaultListColumns) {
		t.Errorf("cols = %v, want %v", cols, datamodel.DefaultListColumns)
	}
	if len(unknown) != 0 {
		t.Errorf("unknown = %v, want none", unknown)
	}
}

func TestFormatItemRowRendersExtras(t *testing.T) {
	cols := []string{"number", "severity", "watchers"}
	set := datamodel.ListItem{
		Number: "KIRA-1",
		Extras: map[string]datamodel.ExtraJSON{"severity": "HIGH", "watchers": []string{"alice", "bob"}},
	}
	if got, want := formatItemRow(cols, set), "KIRA-1\tHIGH\talice,bob"; got != want {
		t.Errorf("set extras: got %q, want %q", got, want)
	}
	absent := datamodel.ListItem{Number: "KIRA-2"}
	if got, want := formatItemRow(cols, absent), "KIRA-2\t-\t-"; got != want {
		t.Errorf("absent extras: got %q, want %q", got, want)
	}
}

func TestWarnUnknownColumns(t *testing.T) {
	var buf bytes.Buffer
	warnUnknownColumns(&buf, []string{"severty"})
	want := "kira: warning: ui.list.columns: unknown column \"severty\"; ignored\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}
