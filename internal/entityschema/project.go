package entityschema

import (
	"slices"
	"strconv"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

// ProjectItem converts an Item into the values map Validate expects. Slices are
// cloned so a caller can't mutate the source Item through the returned map.
func ProjectItem(schema Schema, it *datamodel.Item) map[string]any {
	values := map[string]any{
		"title":      it.Title,
		"state":      it.State,
		"labels":     slices.Clone(it.Labels),
		"blocked_by": slices.Clone(it.BlockedBy),
		"aliases":    slices.Clone(it.Aliases),
		"created":    it.Created,
		"updated":    it.Updated,
	}
	setIfPresent(values, "subtype", it.Subtype)
	setIfPresent(values, "priority", it.Priority)
	setIfPresent(values, "resolution", it.Resolution)
	setIfPresent(values, "rank", it.Rank)
	setIfPresent(values, "owner", it.Owner)
	setIfPresent(values, "reporter", it.Reporter)
	setIfPresent(values, "epic", it.Epic)
	setIfPresent(values, "sprint", it.Sprint)
	setIfPresent(values, "due", it.Due)
	setIfPresent(values, "estimate", it.Estimate)

	sections := bodySections(it.Body)
	for _, f := range schema.Fields {
		switch {
		case f.Placement == PlacementBody && f.Section != "":
			values[f.Name] = sections[f.Section]
		case f.Placement == PlacementFrontmatter:
			setExtra(values, f, it.Extras[f.Name])
		}
	}
	return values
}

func setIfPresent[T any](values map[string]any, name string, v *T) {
	if v != nil {
		values[name] = *v
	}
}

func setExtra(values map[string]any, f FieldDef, v datamodel.ExtraValue) {
	switch {
	case v.List != nil:
		values[f.Name] = decodeElems(v.List)
	case v.Scalar != nil:
		values[f.Name] = typedScalar(f.Type, codec.DecodeScalar(*v.Scalar))
	}
}

func decodeElems(elems []string) []string {
	out := make([]string, len(elems))
	for i, e := range elems {
		out[i] = codec.DecodeScalar(e)
	}
	return out
}

// typedScalar converts an extra's decoded text to the Go type checkType
// expects for f's declared type; unparseable text stays a string so the
// validator reports the mismatch (date/datetime are string-checked as-is).
func typedScalar(t FieldType, s string) any {
	switch t {
	case FieldInt:
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	case FieldNumber:
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	case FieldBool:
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
	}
	return s
}

func bodySections(body string) map[string]string {
	sections := make(map[string]string)
	current := ""
	for line := range strings.SplitSeq(body, "\n") {
		if title, ok := strings.CutPrefix(line, "## "); ok {
			current = title
			continue
		}
		if current != "" {
			sections[current] += line + "\n"
		}
	}
	for title, content := range sections {
		sections[title] = strings.TrimSpace(content)
	}
	return sections
}

// ValidateExtras checks only the frontmatter-extra fields the item type's
// schema declares — builtin fields keep their existing validators, so nothing
// is double-reported. Enum vocabularies resolve through ResolveEnums: inline
// schema enums, overridden by strict config vocab.
func ValidateExtras(schemas map[string]Schema, cfg *datamodel.Config, it *datamodel.Item) []Violation {
	schema, ok := schemas[it.Type]
	if !ok {
		return nil
	}
	extras := extrasOnly(schema)
	if len(extras.Fields) == 0 {
		return nil
	}
	return Validate(extras, ProjectItem(extras, it), ResolveEnums(schema, ConfigVocab(cfg)), nil)
}

func extrasOnly(schema Schema) Schema {
	out := schema
	out.Fields = nil
	for _, f := range schema.Fields {
		if f.Placement == PlacementFrontmatter {
			out.Fields = append(out.Fields, f)
		}
	}
	return out
}

// ConfigVocab mirrors core.validateItem's strict/open policy: a vocab is
// membership-checked only when strict, and the system captured label is always
// admitted under a strict label vocabulary.
func ConfigVocab(cfg *datamodel.Config) map[string][]string {
	enums := map[string][]string{}
	addIfStrict := func(name string, ev datamodel.EnumVocab) {
		if ev.StrictOr(cfg.Labels.Strict) {
			enums[name] = ev.Values
		}
	}
	addIfStrict("priority", cfg.Priorities)
	addIfStrict("subtype", cfg.Subtypes)
	addIfStrict("resolution", cfg.Resolutions)
	if cfg.Labels.Strict {
		enums["label"] = append(slices.Clone(cfg.Labels.Known), datamodel.CapturedLabel)
	}
	return enums
}
