package entityschema

import (
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
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
	it := &datamodel.Item{Priority: ptr.To("P1")}

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
