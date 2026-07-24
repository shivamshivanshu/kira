package codec_test

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestUpsertBodySectionRoundTripsThroughSerializeParse(t *testing.T) {
	it, err := codec.Parse(readExample(t))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	it.Body = datamodel.UpsertBodySection(it.Body, "Description", "rewritten\nsecond line")

	out := codec.Serialize(it)
	reparsed, err := codec.Parse(out)
	if err != nil {
		t.Fatalf("parse(serialize(set)): %v", err)
	}
	if got := datamodel.BodySections(reparsed.Body)["Description"]; got != "rewritten\nsecond line" {
		t.Fatalf("Description = %q, want %q", got, "rewritten\nsecond line")
	}
	if got := datamodel.BodySections(reparsed.Body)["Comments"]; got == "" {
		t.Fatal("unrelated Comments section must survive the edit")
	}
}

// TestUpsertBodySectionReplaceIsByteIdenticalWhenContentUnchanged proves the
// replace path reproduces example.md's exact original bytes when the section
// is set back to the content it already held.
func TestUpsertBodySectionReplaceIsByteIdenticalWhenContentUnchanged(t *testing.T) {
	orig := readExample(t)
	it, err := codec.Parse(orig)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	for _, section := range []string{"Description", "Comments"} {
		content := datamodel.BodySections(it.Body)[section]
		it.Body = datamodel.UpsertBodySection(it.Body, section, content)
	}

	if got := codec.Serialize(it); got != orig {
		t.Fatalf("re-setting a section's own content must be a no-op\n--- got ---\n%s\n--- want ---\n%s", got, orig)
	}
}
