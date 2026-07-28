package entityschema

import (
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

// ProjectItem converts an Item into the values map Validate expects. Slices are
// cloned so a caller can't mutate the source Item through the returned map.
// decodeScalar recovers a declared scalar extra's typed value (e.g.
// codec.DecodeExtraScalar); entityschema stays free of codec/yaml, so callers
// inject it. A ref extra's value is a bare string either way, so it decodes to
// a string regardless — fine for both a typed validator and doctor's
// refStrings.
func ProjectItem(schema Schema, it *datamodel.Item, decodeScalar func(string) any) map[string]any {
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

	sections := datamodel.BodySections(it.Body)
	for _, f := range schema.Fields {
		if f.Placement == PlacementBody && f.Section != "" {
			values[f.Name] = sections[f.Section]
		}
	}

	for k, v := range it.ExtraScalars {
		values[k] = decodeScalar(v)
	}
	for k, v := range it.ExtraLists {
		values[k] = slices.Clone(v)
	}
	return values
}

// CustomFields projects the fields a schema adds beyond the built-in item
// shape, and is nil for a schema that adds none.
func CustomFields(schema Schema, it *datamodel.Item, decodeScalar func(string) any) map[string]any {
	var customValues map[string]any
	allValues := ProjectItem(schema, it, decodeScalar)
	for _, f := range schema.Fields {
		if !f.isCustomFrontmatterField() {
			continue
		}
		value, present := allValues[f.Name]
		if !present {
			continue
		}
		if customValues == nil {
			customValues = map[string]any{}
		}
		customValues[f.Name] = value
	}
	return customValues
}

func (f FieldDef) isCustomFrontmatterField() bool {
	if f.System || f.Placement == PlacementBody {
		return false
	}
	_, hasDedicatedItemField := datamodel.Field(f.Name)
	return !hasDedicatedItemField
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
