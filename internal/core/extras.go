package core

import (
	"maps"
	"slices"

	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/setx"
)

// declaredExtras lists every schema-declared frontmatter extra, deduped by
// name across schemas (first declaration wins) in schema-name then field
// order. Callers that key off a file path rather than an item use the union
// because they cannot know the item's type.
func declaredExtras(schemas map[string]entityschema.Schema) []entityschema.FieldDef {
	seen := setx.NewDeduper[string]()
	var out []entityschema.FieldDef
	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		for _, f := range schemas[name].Fields {
			if !f.IsCustomFrontmatter() || !seen.Add(f.Name) {
				continue
			}
			out = append(out, f)
		}
	}
	return out
}
