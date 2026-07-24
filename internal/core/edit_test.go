package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func extraFieldSchema(t *testing.T) entityschema.Schema {
	t.Helper()
	schema := ticketSchema(t)
	schema.Enums = append(schema.Enums, entityschema.EnumDef{Name: "severity", Values: []string{"low", "high"}})
	schema.Fields = append(schema.Fields,
		entityschema.FieldDef{Name: "severity", Type: entityschema.FieldEnum, Enum: "severity"},
		entityschema.FieldDef{Name: "points", Type: entityschema.FieldInt},
		entityschema.FieldDef{Name: "due_custom", Type: entityschema.FieldDate},
		entityschema.FieldDef{Name: "tags", Type: entityschema.FieldString, List: true},
		entityschema.FieldDef{Name: "notes", Type: entityschema.FieldMarkdown, Placement: entityschema.PlacementBody, Section: "Notes"},
	)
	return schema
}

func TestApplyFieldEditDeclaredExtraScalar(t *testing.T) {
	schema := extraFieldSchema(t)
	it := &datamodel.Item{}

	if err := applyFieldEdit(it, "severity", "high", schema); err != nil {
		t.Fatalf("set enum extra: %v", err)
	}
	if it.ExtraScalars["severity"] != "high" {
		t.Errorf("ExtraScalars[severity] = %q, want %q", it.ExtraScalars["severity"], "high")
	}

	if err := applyFieldEdit(it, "points", "3", schema); err != nil {
		t.Fatalf("set int extra: %v", err)
	}
	if it.ExtraScalars["points"] != "3" {
		t.Errorf("ExtraScalars[points] = %q, want %q", it.ExtraScalars["points"], "3")
	}

	if err := applyFieldEdit(it, "points", "abc", schema); err == nil {
		t.Error("a non-int value must be rejected for a declared int extra")
	}

	if err := applyFieldEdit(it, "severity", "", schema); err != nil {
		t.Fatalf("clear scalar extra: %v", err)
	}
	if _, ok := it.ExtraScalars["severity"]; ok {
		t.Error("--field key= must clear the extra")
	}
}

func TestApplyFieldEditDeclaredExtraDateRoundTripsAndValidates(t *testing.T) {
	schema := extraFieldSchema(t)
	it := &datamodel.Item{
		ID: "X", Number: "KIRA-1", Type: datamodel.TypeTicket, Title: "t", State: "TODO",
		Aliases: []string{}, Created: "2026-07-20T10:00:00Z", Updated: "2026-07-20T10:00:00Z",
	}
	if err := applyFieldEdit(it, "due_custom", "2026-07-24", schema); err != nil {
		t.Fatalf("set date extra: %v", err)
	}
	if it.ExtraScalars["due_custom"] != `"2026-07-24"` {
		t.Errorf("ExtraScalars[due_custom] = %q, want a quoted date so it decodes back as a string", it.ExtraScalars["due_custom"])
	}
	errs, _ := newValidationCtx(config.Default(), nil, nil, schema, false).validateItem(it)
	for _, e := range errs {
		if strings.Contains(e.Error(), "due_custom") {
			t.Errorf("declared date extra must validate, got: %v", e)
		}
	}
}

func TestApplyFieldEditDeclaredExtraList(t *testing.T) {
	schema := extraFieldSchema(t)
	it := &datamodel.Item{}

	if err := applyFieldEdit(it, "tags", "a, b", schema); err != nil {
		t.Fatalf("set list extra: %v", err)
	}
	want := []string{"a", "b"}
	if got := it.ExtraLists["tags"]; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("ExtraLists[tags] = %v, want %v", got, want)
	}

	if err := applyFieldEdit(it, "tags", "", schema); err != nil {
		t.Fatalf("clear list extra: %v", err)
	}
	if _, ok := it.ExtraLists["tags"]; ok {
		t.Error("--field key= must clear the list extra")
	}
}

func TestApplyFieldEditSystemFieldRejected(t *testing.T) {
	schema := extraFieldSchema(t)
	it := &datamodel.Item{Created: "2026-07-20T10:00:00Z"}
	if err := applyFieldEdit(it, "created", "2020-01-01T00:00:00Z", schema); err == nil {
		t.Fatal("a system schema field must not be settable via --field")
	}
	if _, ok := it.ExtraScalars["created"]; ok {
		t.Error("a system field must never leak into ExtraScalars (duplicate frontmatter key)")
	}
}

func TestApplyFieldEditBodyPlacedFieldRejected(t *testing.T) {
	schema := extraFieldSchema(t)
	it := &datamodel.Item{}
	if err := applyFieldEdit(it, "notes", "x", schema); err == nil {
		t.Error("a body-placed declared field must not be settable via --field yet")
	}
}

func TestApplyFieldEditUnknownKeySuggestsDeclaredExtra(t *testing.T) {
	schema := extraFieldSchema(t)
	it := &datamodel.Item{}
	err := applyFieldEdit(it, "severty", "high", schema)
	if err == nil {
		t.Fatal("unknown key must be rejected")
	}
	var e *errx.Error
	if !errors.As(err, &e) {
		t.Fatalf("want *errx.Error, got %T", err)
	}
	if !strings.Contains(e.Hint, "severity") {
		t.Errorf("hint = %q, want a did-you-mean for the declared extra field severity", e.Hint)
	}
}
