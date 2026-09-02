package entityschema

import (
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

// ExtraDecoder recovers the values behind the rendered YAML text that
// Item.ExtraScalars and Item.ExtraLists hold. It is an interface rather than a
// codec import because entityschema sits below codec, and Go's structural
// typing means codec never has to name this type. List elements decode to
// strings because a list field is string-valued by schema rule, which is what
// the validator and doctor's refStrings both expect.
type ExtraDecoder interface {
	DecodeExtraScalar(datamodel.Rendered) any
	DecodeExtraList([]datamodel.Rendered) []string
}

// ProjectItem converts an Item into the values map Validate expects. Slices are
// cloned so a caller can't mutate the source Item through the returned map.
func ProjectItem(schema Schema, it *datamodel.Item, dec ExtraDecoder) map[string]any {
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

	var sections map[string]string
	for _, f := range schema.Fields {
		if f.Placement == PlacementBody && f.Section != "" {
			if sections == nil {
				sections = datamodel.BodySections(it.Body)
			}
			values[f.Name] = sections[f.Section]
		}
	}

	for k, v := range it.ExtraScalars {
		values[k] = dec.DecodeExtraScalar(v)
	}
	for k, v := range it.ExtraLists {
		values[k] = dec.DecodeExtraList(v)
	}
	return values
}

// CustomFields projects the fields a schema adds beyond the built-in item
// shape, and is nil for a schema that adds none. It reads the extras directly
// rather than going through ProjectItem: this runs per row on every listing,
// and ProjectItem parses the whole body to build a map whose built-in keys are
// all discarded here.
func CustomFields(schema Schema, it *datamodel.Item, dec ExtraDecoder) map[string]any {
	var customValues map[string]any
	put := func(name string, value any) {
		if customValues == nil {
			customValues = map[string]any{}
		}
		customValues[name] = value
	}
	for _, f := range schema.Fields {
		if !f.IsCustomFrontmatter() {
			continue
		}
		if f.List {
			if list, ok := it.ExtraLists[f.Name]; ok {
				put(f.Name, dec.DecodeExtraList(list))
			}
			continue
		}
		if raw, ok := it.ExtraScalars[f.Name]; ok {
			put(f.Name, dec.DecodeExtraScalar(raw))
		}
	}
	return customValues
}

// IsSchemaOnly reports whether a schema declares this field beyond the built-in
// item shape: no dedicated Item field behind it and not system-managed, whether
// it lands in frontmatter or a body section.
func (f FieldDef) IsSchemaOnly() bool {
	if f.System {
		return false
	}
	_, hasDedicatedItemField := datamodel.Field(f.Name)
	return !hasDedicatedItemField
}

// IsCustomFrontmatter narrows IsSchemaOnly to the fields whose value lives in
// the item's extras, which is what can be read back and filtered; a body-placed
// field's text reaches callers through the body instead.
func (f FieldDef) IsCustomFrontmatter() bool {
	return f.IsSchemaOnly() && f.Placement != PlacementBody
}

// ValidateExtras checks only the frontmatter-extra fields the item type's
// schema declares — builtin fields keep their existing validators, so nothing
// is double-reported. Enum vocabularies resolve through ResolveEnums: inline
// schema enums, overridden by config vocab.
func ValidateExtras(schemas map[string]Schema, cfg *datamodel.Config, it *datamodel.Item, dec ExtraDecoder) []Violation {
	schema, ok := schemas[it.Type]
	if !ok {
		return nil
	}
	extras := extrasOnly(schema)
	if len(extras.Fields) == 0 {
		return nil
	}
	return Validate(extras, ProjectItem(extras, it, dec), ResolveEnums(schema, ConfigVocab(cfg)), nil)
}

func extrasOnly(schema Schema) Schema {
	out := schema
	out.Fields = nil
	for _, f := range schema.Fields {
		if f.IsCustomFrontmatter() {
			out.Fields = append(out.Fields, f)
		}
	}
	return out
}

func setIfPresent[T any](values map[string]any, name string, v *T) {
	if v != nil {
		values[name] = *v
	}
}

// ConfigVocab surfaces each config-driven vocabulary's known values (the system
// captured label always admitted). label and person are always membership-
// checked (empty Known means "nothing known yet", not "unchecked"); priority/
// subtype/resolution are free-form when their config leaves Values empty.
// Strictness (error vs warning) is the caller's policy, not this function's.
func ConfigVocab(cfg *datamodel.Config) map[string][]string {
	enums := map[string][]string{
		"label":  append(slices.Clone(cfg.Labels.Known), datamodel.CapturedLabel),
		"person": cfg.People.Names(),
	}
	addIfConfigured := func(name string, ev datamodel.EnumVocab) {
		if len(ev.Values) > 0 {
			enums[name] = ev.Values
		}
	}
	addIfConfigured("priority", cfg.Priorities)
	addIfConfigured("subtype", cfg.Subtypes)
	addIfConfigured("resolution", cfg.Resolutions)
	return enums
}
