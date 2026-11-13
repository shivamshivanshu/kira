package datamodel

import "slices"

// BodyField is a markdown body section addressable as an item field, so
// `--field description=...` edits the "## Description" section in place.
type BodyField struct {
	Field   string
	Section string
}

var BodyFields = []BodyField{
	{Field: "description", Section: "Description"},
	{Field: "acceptance", Section: "Acceptance criteria"},
}

func LookupBodyField(field string) (BodyField, bool) {
	i := slices.IndexFunc(BodyFields, func(f BodyField) bool { return f.Field == field })
	if i < 0 {
		return BodyField{}, false
	}
	return BodyFields[i], true
}
