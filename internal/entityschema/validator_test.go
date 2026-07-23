package entityschema

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func testSchema() Schema {
	return Schema{
		Name: "widget",
		Fields: []FieldDef{
			{Name: "title", Type: FieldString, Required: true},
			{Name: "priority", Type: FieldEnum, Enum: "priority"},
			{Name: "labels", Type: FieldEnum, Enum: "label", List: true, Unique: true},
			{Name: "count", Type: FieldInt},
			{Name: "owner", Type: FieldRef, Target: "person"},
			{Name: "state", Type: FieldState, States: []StateValue{
				{Key: "TODO", Category: datamodel.CategoryTodo},
				{Key: "DONE", Category: datamodel.CategoryDone},
			}},
		},
	}
}

func TestValidateAcceptsConformingValues(t *testing.T) {
	schema := testSchema()
	values := map[string]any{
		"title":    "a widget",
		"priority": "P1",
		"labels":   []string{"core", "tui"},
		"count":    3,
		"owner":    "shivam",
	}
	enums := map[string][]string{"priority": {"P0", "P1", "P2"}}

	violations := Validate(schema, values, enums, nil)

	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateMissingRequiredField(t *testing.T) {
	schema := testSchema()
	values := map[string]any{"priority": "P1"}

	violations := Validate(schema, values, nil, nil)

	if len(violations) != 1 || violations[0].Field != "title" {
		t.Fatalf("expected exactly one violation on %q, got %v", "title", violations)
	}
}

func TestValidateEnumNotInVocabulary(t *testing.T) {
	schema := testSchema()
	values := map[string]any{"title": "x", "priority": "P9"}
	enums := map[string][]string{"priority": {"P0", "P1", "P2"}}

	violations := Validate(schema, values, enums, nil)

	if len(violations) != 1 || violations[0].Field != "priority" {
		t.Fatalf("expected exactly one violation on %q, got %v", "priority", violations)
	}
}

func TestValidateEnumOpenWhenVocabularyOmitted(t *testing.T) {
	schema := testSchema()
	values := map[string]any{"title": "x", "priority": "anything"}

	violations := Validate(schema, values, nil, nil)

	if len(violations) != 0 {
		t.Fatalf("expected no violations for an unsupplied vocabulary, got %v", violations)
	}
}

func TestValidateWrongScalarType(t *testing.T) {
	schema := testSchema()
	values := map[string]any{"title": "x", "count": "three"}

	violations := Validate(schema, values, nil, nil)

	if len(violations) != 1 || violations[0].Field != "count" {
		t.Fatalf("expected exactly one violation on %q, got %v", "count", violations)
	}
}

func TestValidateListElementNotAList(t *testing.T) {
	schema := testSchema()
	values := map[string]any{"title": "x", "labels": "core"}

	violations := Validate(schema, values, nil, nil)

	if len(violations) != 1 || violations[0].Field != "labels" {
		t.Fatalf("expected exactly one violation on %q, got %v", "labels", violations)
	}
}

func TestValidateDuplicateInUniqueList(t *testing.T) {
	schema := testSchema()
	values := map[string]any{"title": "x", "labels": []string{"core", "core"}}

	violations := Validate(schema, values, nil, nil)

	if len(violations) != 1 || violations[0].Field != "labels" {
		t.Fatalf("expected exactly one violation on %q, got %v", "labels", violations)
	}
}

func TestValidateRefExistenceSkippedWithoutResolver(t *testing.T) {
	schema := testSchema()
	values := map[string]any{"title": "x", "owner": "nobody"}

	violations := Validate(schema, values, nil, nil)

	if len(violations) != 0 {
		t.Fatalf("Phase 1 defers ref existence checks, got %v", violations)
	}
}

type stubResolver struct{ known map[string]bool }

func (r stubResolver) Exists(target, id string) bool { return r.known[target+":"+id] }

func TestValidateRefExistenceEnforcedWithResolver(t *testing.T) {
	schema := testSchema()
	values := map[string]any{"title": "x", "owner": "nobody"}
	refs := stubResolver{known: map[string]bool{"person:shivam": true}}

	violations := Validate(schema, values, nil, refs)

	if len(violations) != 1 || violations[0].Field != "owner" {
		t.Fatalf("expected exactly one violation on %q, got %v", "owner", violations)
	}
}

func TestValidateStateInDeclaredSet(t *testing.T) {
	schema := testSchema()
	values := map[string]any{"title": "x", "state": "DONE"}

	violations := Validate(schema, values, nil, nil)

	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateStateNotDeclared(t *testing.T) {
	schema := testSchema()
	values := map[string]any{"title": "x", "state": "IN_REVIEW"}

	violations := Validate(schema, values, nil, nil)

	if len(violations) != 1 || violations[0].Field != "state" {
		t.Fatalf("expected exactly one violation on %q, got %v", "state", violations)
	}
}

func TestValidateAcceptsWellFormedDateAndDatetime(t *testing.T) {
	schema := Schema{Name: "w", Fields: []FieldDef{
		{Name: "due", Type: FieldDate},
		{Name: "created", Type: FieldDatetime},
	}}
	values := map[string]any{"due": "2026-07-23", "created": "2026-07-23T15:00:00+05:30"}

	violations := Validate(schema, values, nil, nil)

	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %v", violations)
	}
}

func TestValidateRejectsMalformedDateAndDatetime(t *testing.T) {
	schema := Schema{Name: "w", Fields: []FieldDef{
		{Name: "due", Type: FieldDate},
		{Name: "created", Type: FieldDatetime},
	}}
	values := map[string]any{"due": "2026-13-99", "created": "not-a-time"}

	violations := Validate(schema, values, nil, nil)

	if len(violations) != 2 {
		t.Fatalf("expected both malformed temporal fields flagged, got %v", violations)
	}
}

func TestValidateAcceptsCapturedLabelUnderStrictConfig(t *testing.T) {
	schema := Schema{Name: "ticket", Fields: []FieldDef{
		{Name: "labels", Type: FieldEnum, Enum: "label", List: true},
	}}
	cfg := &datamodel.Config{Labels: datamodel.Vocab{Known: []string{"core"}, Strict: true}}
	values := map[string]any{"labels": []string{"core", datamodel.CapturedLabel}}

	violations := Validate(schema, values, ConfigVocab(cfg), nil)

	if len(violations) != 0 {
		t.Fatalf("system captured label must pass a strict label vocabulary, got %v", violations)
	}
}

func TestResolveEnumsConfigVocabWinsOverInline(t *testing.T) {
	schema := Schema{Name: "widget", Enums: []EnumDef{{Name: "priority", Values: []string{"stale"}}}}

	resolved := ResolveEnums(schema, map[string][]string{"priority": {"P0", "P1"}})

	if len(resolved["priority"]) != 2 || resolved["priority"][0] != "P0" {
		t.Fatalf("config vocab should win on a name conflict, got %v", resolved["priority"])
	}
}
