package entityschema

import (
	"slices"
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
