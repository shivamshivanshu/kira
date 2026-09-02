package doctor

import (
	"fmt"
	"maps"
	"slices"

	"github.com/charmbracelet/x/ansi"

	"github.com/shivamshivanshu/kira/internal/entityschema"
)

// A type glyph shares one column with every other type's, sized to the widest
// of them, so an icon far wider than a glyph widens that column for every row.
// The threshold sits above a bracketed ascii glyph ("[T]", three cells, a
// deliberate and reasonable choice) and below every stale field name that
// predates representation.icon holding a literal glyph -- "epic", "board" and
// "ticket" are four cells or more, still parse, and still render, as the word.
const maxReasonableIconCells = 3

func SchemaIcons(schemas map[string]entityschema.Schema) []Finding {
	var out []Finding
	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		icon := schemas[name].Representation.Icon
		cells := ansi.StringWidth(icon)
		if cells <= maxReasonableIconCells {
			continue
		}
		out = append(out, Finding{
			Class:    ClassSchema,
			Severity: SeverityWarning,
			Message: fmt.Sprintf("schema %q declares a %d-cell representation icon %q, which widens the type column for every row; a glyph is normally 1-3 cells",
				name, cells, icon),
		})
	}
	return out
}
