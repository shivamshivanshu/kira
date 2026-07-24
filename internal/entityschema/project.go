package entityschema

import (
	"slices"
	"strings"

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
		if f.Placement == PlacementBody && f.Section != "" {
			values[f.Name] = sections[f.Section]
		}
	}
	return values
}

func setIfPresent[T any](values map[string]any, name string, v *T) {
	if v != nil {
		values[name] = *v
	}
}

func bodySections(body string) map[string]string {
	sections := make(map[string]string)
	current := ""
	for _, line := range strings.Split(body, "\n") {
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
