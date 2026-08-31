package entityschema

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestBodySectionsPreservesRepeatedHeadings(t *testing.T) {
	body := "ignored\n## Description\n first\n## Other\nelse\n## Description\n second\n"
	schema := Schema{Fields: []FieldDef{{Name: "description", Placement: PlacementBody, Section: "Description"}}}
	got := ProjectItem(schema, &datamodel.Item{Body: body})
	if got["description"] != "first\n second" {
		t.Fatalf("repeated section = %q", got["description"])
	}
	got = ProjectItem(schema, &datamodel.Item{Body: "## Description\n" + strings.Repeat("line\n", 10000)})
	if got["description"] != strings.TrimSpace(strings.Repeat("line\n", 10000)) {
		t.Fatal("long section lost content")
	}
}
