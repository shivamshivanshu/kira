package entityschema

import (
	"slices"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/ptr"
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

	values := ProjectItem(schema, it)

	if values["notes"] != "some prose" {
		t.Fatalf("got %q, want %q", values["notes"], "some prose")
	}
}

func TestProjectItemOmitsUnsetOptionalScalars(t *testing.T) {
	it := &datamodel.Item{Title: "x"}

	values := ProjectItem(Schema{}, it)

	if _, present := values["priority"]; present {
		t.Fatalf("unset priority should be absent, got %v", values["priority"])
	}
}

func TestProjectItemIncludesPresentOptionalScalars(t *testing.T) {
	it := &datamodel.Item{Priority: ptr.To("P1")}

	values := ProjectItem(Schema{}, it)

	if values["priority"] != "P1" {
		t.Fatalf("got %v, want %q", values["priority"], "P1")
	}
}

func TestProjectItemTypesDeclaredExtras(t *testing.T) {
	schema := Schema{
		Name: "ticket",
		Fields: []FieldDef{
			{Name: "points", Type: FieldInt, Placement: PlacementFrontmatter},
			{Name: "tags", Type: FieldString, List: true, Placement: PlacementFrontmatter},
		},
	}
	it := &datamodel.Item{Extras: map[string]datamodel.ExtraValue{
		"points": {Scalar: ptr.To("42")},
		"tags":   {List: []string{"a", `"b c"`}},
	}}

	values := ProjectItem(schema, it)

	if values["points"] != int64(42) {
		t.Fatalf("points = %v (%T), want int64 42", values["points"], values["points"])
	}
	if got, want := values["tags"].([]string), []string{"a", "b c"}; !slices.Equal(got, want) {
		t.Fatalf("tags = %v, want %v", got, want)
	}
	if v := Validate(schema, values, nil, nil); len(v) != 0 {
		t.Fatalf("expected no violations, got %v", v)
	}
}

func TestValidateFlagsMistypedDeclaredExtra(t *testing.T) {
	schema := Schema{
		Name:   "ticket",
		Fields: []FieldDef{{Name: "points", Type: FieldInt, Placement: PlacementFrontmatter}},
	}
	it := &datamodel.Item{Extras: map[string]datamodel.ExtraValue{"points": {Scalar: ptr.To("abc")}}}

	v := Validate(schema, ProjectItem(schema, it), nil, nil)

	if len(v) != 1 || v[0].Field != "points" {
		t.Fatalf("expected one points type violation, got %v", v)
	}
}

func TestConfigVocabOmitsNonStrictVocab(t *testing.T) {
	cfg := &datamodel.Config{
		Labels:     datamodel.Vocab{Strict: false},
		Priorities: datamodel.EnumVocab{Values: []string{"P0", "P1"}},
	}

	enums := ConfigVocab(cfg)

	if _, present := enums["priority"]; present {
		t.Fatalf("non-strict vocab should be open (absent), got %v", enums["priority"])
	}
}

func TestConfigVocabIncludesStrictVocab(t *testing.T) {
	strict := true
	cfg := &datamodel.Config{
		Labels:     datamodel.Vocab{Strict: false},
		Priorities: datamodel.EnumVocab{Values: []string{"P0", "P1"}, Strict: &strict},
	}

	enums := ConfigVocab(cfg)

	if len(enums["priority"]) != 2 {
		t.Fatalf("expected the strict vocab to be enforced, got %v", enums["priority"])
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

func TestValidateExtrasEnforcesDeclaredTypes(t *testing.T) {
	schemas := map[string]Schema{"ticket": {
		Name:  "ticket",
		Enums: []EnumDef{{Name: "sev", Values: []string{"LOW", "HIGH"}}},
		Fields: []FieldDef{
			{Name: "title", Type: FieldString, Required: true},
			{Name: "severity", Type: FieldEnum, Enum: "sev", Placement: PlacementFrontmatter},
			{Name: "story_points", Type: FieldInt, Placement: PlacementFrontmatter},
			{Name: "ratio", Type: FieldNumber, Placement: PlacementFrontmatter},
			{Name: "urgent", Type: FieldBool, Placement: PlacementFrontmatter},
			{Name: "starts", Type: FieldDate, Placement: PlacementFrontmatter},
			{Name: "seen", Type: FieldDatetime, Placement: PlacementFrontmatter},
			{Name: "tags", Type: FieldString, List: true, Placement: PlacementFrontmatter},
		},
	}}
	cfg := &datamodel.Config{}

	valid := &datamodel.Item{Type: "ticket", Extras: map[string]datamodel.ExtraValue{
		"severity":     {Scalar: ptr.To("LOW")},
		"story_points": {Scalar: ptr.To("3")},
		"ratio":        {Scalar: ptr.To("1.5")},
		"urgent":       {Scalar: ptr.To("true")},
		"starts":       {Scalar: ptr.To("2026-01-02")},
		"seen":         {Scalar: ptr.To("2026-01-02T03:04:05Z")},
		"tags":         {List: []string{"a", "b"}},
		"undeclared":   {Scalar: ptr.To("ignored")},
	}}
	if v := ValidateExtras(schemas, cfg, valid); len(v) != 0 {
		t.Fatalf("conforming extras must not violate (and builtin title must stay unchecked), got %v", v)
	}

	invalid := []struct {
		field string
		value datamodel.ExtraValue
		want  string
	}{
		{"severity", datamodel.ExtraValue{Scalar: ptr.To("BOGUS")}, `not in the "sev" vocabulary`},
		{"story_points", datamodel.ExtraValue{Scalar: ptr.To("abc")}, "expected type int"},
		{"ratio", datamodel.ExtraValue{Scalar: ptr.To("fast")}, "expected type number"},
		{"urgent", datamodel.ExtraValue{Scalar: ptr.To("maybe")}, "expected type bool"},
		{"starts", datamodel.ExtraValue{Scalar: ptr.To("not-a-date")}, "expected type date"},
		{"seen", datamodel.ExtraValue{Scalar: ptr.To("2026-13-99")}, "expected type datetime"},
		{"story_points", datamodel.ExtraValue{List: []string{"3"}}, "expected type int"},
		{"tags", datamodel.ExtraValue{Scalar: ptr.To("a")}, "expected a list"},
	}
	for _, tt := range invalid {
		it := &datamodel.Item{Type: "ticket", Extras: map[string]datamodel.ExtraValue{tt.field: tt.value}}
		v := ValidateExtras(schemas, cfg, it)
		if len(v) != 1 || v[0].Field != tt.field || !strings.Contains(v[0].Message, tt.want) {
			t.Errorf("%s = %v: expected one violation containing %q", tt.field, v, tt.want)
		}
	}

	noSchema := &datamodel.Item{Type: "gadget", Extras: map[string]datamodel.ExtraValue{"x": {Scalar: ptr.To("y")}}}
	if v := ValidateExtras(schemas, cfg, noSchema); v != nil {
		t.Fatalf("a type without a schema has nothing declared to validate, got %v", v)
	}
}
