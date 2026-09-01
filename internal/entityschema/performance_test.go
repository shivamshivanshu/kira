package entityschema

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestProjectItemKeepsLongBodySectionIntact(t *testing.T) {
	schema := Schema{Fields: []FieldDef{{Name: "description", Placement: PlacementBody, Section: "Description"}}}
	body := "## Description\n" + strings.Repeat("line\n", 10000)

	got := ProjectItem(schema, &datamodel.Item{Body: body}, codec.DecodeExtraScalar)

	if got["description"] != strings.TrimSpace(strings.Repeat("line\n", 10000)) {
		t.Fatal("long section lost content")
	}
}
