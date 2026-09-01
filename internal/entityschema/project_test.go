package entityschema

import (
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestProjectItemReadsMarkdownFromItsDeclaredSection(t *testing.T) {
	schema := Schema{
		Name: "ticket",
		Fields: []FieldDef{
			{Name: "notes", Type: FieldMarkdown, Placement: PlacementBody, Section: "Notes"},
		},
	}
	it := &datamodel.Item{
		Body: "## Notes\nsome prose\n\n## Comments\n<!-- kira:comment id=x author=a ts=t -->\nhi\n<!-- /kira:comment -->\n",
	}

	values := ProjectItem(schema, it, codec.DecodeExtraScalar)

	if values["notes"] != "some prose" {
		t.Fatalf("got %q, want %q", values["notes"], "some prose")
	}
}

func TestRequiredBodyFieldValidatesPresentAndErrorsAbsent(t *testing.T) {
	schema := Schema{
		Name: "ticket",
		Fields: []FieldDef{
			{Name: "notes", Type: FieldMarkdown, Required: true, Placement: PlacementBody, Section: "Notes"},
		},
	}

	present := &datamodel.Item{Body: "## Notes\nsome prose\n"}
	if violations := Validate(schema, ProjectItem(schema, present, codec.DecodeExtraScalar), nil, nil); len(violations) != 0 {
		t.Fatalf("a present required body field must validate, got %v", violations)
	}

	absent := &datamodel.Item{Body: ""}
	violations := Validate(schema, ProjectItem(schema, absent, codec.DecodeExtraScalar), nil, nil)
	if len(violations) != 1 || violations[0].Field != "notes" || violations[0].Code != CodeRequired {
		t.Fatalf("an absent required body field must error required, got %v", violations)
	}
}

func TestProjectItemOmitsUnsetOptionalScalars(t *testing.T) {
	it := &datamodel.Item{Title: "x"}

	values := ProjectItem(Schema{}, it, codec.DecodeExtraScalar)

	if _, present := values["priority"]; present {
		t.Fatalf("unset priority should be absent, got %v", values["priority"])
	}
}

func TestProjectItemIncludesPresentOptionalScalars(t *testing.T) {
	it := &datamodel.Item{Priority: new("P1")}

	values := ProjectItem(Schema{}, it, codec.DecodeExtraScalar)

	if values["priority"] != "P1" {
		t.Fatalf("got %v, want %q", values["priority"], "P1")
	}
}

func TestConfigVocabOmitsEmptyVocab(t *testing.T) {
	cfg := &datamodel.Config{Priorities: datamodel.EnumVocab{}}

	enums := ConfigVocab(cfg)

	if _, present := enums["priority"]; present {
		t.Fatalf("a priority vocab with no configured values should be free-form (absent), got %v", enums["priority"])
	}
}

// Strictness is not ConfigVocab's concern: it only decides whether an
// out-of-vocabulary value is an error or a warning, a call the caller makes
// after Validate runs (see core.enumStrict), not whether membership is
// checked at all.
func TestConfigVocabIncludesConfiguredVocabRegardlessOfStrict(t *testing.T) {
	cfg := &datamodel.Config{Priorities: datamodel.EnumVocab{Values: []string{"P0", "P1"}}}

	enums := ConfigVocab(cfg)

	if len(enums["priority"]) != 2 {
		t.Fatalf("a configured vocab must be membership-checked even when non-strict, got %v", enums["priority"])
	}
}

func TestConfigVocabAdmitsCapturedLabelUnderStrict(t *testing.T) {
	cfg := &datamodel.Config{
		Labels: datamodel.Vocab{Known: []string{"area"}, Strict: true},
	}

	enums := ConfigVocab(cfg)

	if !slices.Contains(enums["label"], datamodel.CapturedLabel) {
		t.Fatalf("strict label vocab must admit the system %q label, got %v", datamodel.CapturedLabel, enums["label"])
	}
}

func TestProjectItemTypesDeclaredExtras(t *testing.T) {
	schema := Schema{
		Name: "ticket",
		Fields: []FieldDef{
			{Name: "points", Type: FieldInt},
			{Name: "tags", Type: FieldString, List: true},
		},
	}
	it := &datamodel.Item{
		ExtraScalars: map[string]string{"points": "42"},
		ExtraLists:   map[string][]string{"tags": {"a", "b c"}},
	}

	values := ProjectItem(schema, it, codec.DecodeExtraScalar)

	if !isWholeNumber(values["points"]) {
		t.Fatalf("points = %v (%T), want a whole number", values["points"], values["points"])
	}
	if got, want := values["tags"].([]string), []string{"a", "b c"}; !slices.Equal(got, want) {
		t.Fatalf("tags = %v, want %v", got, want)
	}
	if v := Validate(schema, values, nil, nil); len(v) != 0 {
		t.Fatalf("expected no violations, got %v", v)
	}
}

func TestValidateExtrasEnforcesDeclaredTypes(t *testing.T) {
	schemas := map[string]Schema{"ticket": {
		Name:  "ticket",
		Enums: []EnumDef{{Name: "sev", Values: []string{"LOW", "HIGH"}}},
		Fields: []FieldDef{
			{Name: "title", Type: FieldString, Required: true},
			{Name: "severity", Type: FieldEnum, Enum: "sev"},
			{Name: "story_points", Type: FieldInt},
			{Name: "ratio", Type: FieldNumber},
			{Name: "urgent", Type: FieldBool},
			{Name: "starts", Type: FieldDate},
			{Name: "seen", Type: FieldDatetime},
			{Name: "tags", Type: FieldString, List: true},
		},
	}}
	cfg := &datamodel.Config{}

	valid := &datamodel.Item{
		Type: "ticket",
		ExtraScalars: map[string]string{
			"severity":     "LOW",
			"story_points": "3",
			"ratio":        "1.5",
			"urgent":       "true",
			"starts":       "2026-01-02",
			"seen":         "2026-01-02T03:04:05Z",
			"undeclared":   "ignored",
		},
		ExtraLists: map[string][]string{"tags": {"a", "b"}},
	}
	if v := ValidateExtras(schemas, cfg, valid, codec.DecodeExtraScalar); len(v) != 0 {
		t.Fatalf("conforming extras must not violate (and builtin title must stay unchecked), got %v", v)
	}

	invalid := []struct {
		field  string
		scalar string
		list   []string
		want   string
	}{
		{field: "severity", scalar: "BOGUS", want: `not in the "sev" vocabulary`},
		{field: "story_points", scalar: "abc", want: "expected type int"},
		{field: "ratio", scalar: "fast", want: "expected type number"},
		{field: "urgent", scalar: "maybe", want: "expected type bool"},
		{field: "starts", scalar: "not-a-date", want: "expected type date"},
		{field: "seen", scalar: "2026-13-99", want: "expected type datetime"},
		{field: "story_points", list: []string{"3"}, want: "expected type int"},
		{field: "tags", scalar: "a", want: "expected a list"},
	}
	for _, tt := range invalid {
		it := &datamodel.Item{Type: "ticket"}
		if tt.list != nil {
			it.ExtraLists = map[string][]string{tt.field: tt.list}
		} else {
			it.ExtraScalars = map[string]string{tt.field: tt.scalar}
		}
		v := ValidateExtras(schemas, cfg, it, codec.DecodeExtraScalar)
		if len(v) != 1 || v[0].Field != tt.field || !strings.Contains(v[0].Message, tt.want) {
			t.Errorf("%s = %v: expected one violation containing %q", tt.field, v, tt.want)
		}
	}

	noSchema := &datamodel.Item{Type: "gadget", ExtraScalars: map[string]string{"x": "y"}}
	if v := ValidateExtras(schemas, cfg, noSchema, codec.DecodeExtraScalar); v != nil {
		t.Fatalf("a type without a schema has nothing declared to validate, got %v", v)
	}
}

func TestTargetsItemTypeExcludesConfigVocabularyRefs(t *testing.T) {
	itemTypes := map[string]bool{"ticket": true, "epic": true}

	if !(FieldDef{Type: FieldRef, Target: "epic"}).TargetsItemType(itemTypes) {
		t.Fatal("a plain ref to an item type must be an item reference")
	}
	if (FieldDef{Type: FieldRef, Target: "person", Source: "people"}).TargetsItemType(itemTypes) {
		t.Fatal("a source-bearing ref names a config vocabulary, not an item")
	}
	if (FieldDef{Type: FieldRef, Target: "ticket", Source: "people"}).TargetsItemType(itemTypes) {
		t.Fatal("a source-bearing ref stays a vocabulary reference even when its target names an item type")
	}
}
