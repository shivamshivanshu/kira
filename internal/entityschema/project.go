package entityschema

import (
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

// ProjectItem converts an Item into the values map Validate expects.
// Frontmatter scalars map directly off Item's fixed struct fields, since
// Item's shape predates this package (a later phase may replace it).
// Markdown fields are read out of the body by the schema's own declared
// Placement/Section, not a hardcoded section list, so a schema stays the
// single source of truth for where its content lives.
func ProjectItem(schema Schema, it *datamodel.Item) map[string]any {
	values := map[string]any{
		"title":      it.Title,
		"state":      it.State,
		"labels":     it.Labels,
		"blocked_by": it.BlockedBy,
		"aliases":    it.Aliases,
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
	if it.Estimate != nil {
		values["estimate"] = *it.Estimate
	}

	sections := bodySections(it.Body)
	for _, f := range schema.Fields {
		if f.Placement == PlacementBody && f.Section != "" {
			values[f.Name] = sections[f.Section]
		}
	}
	return values
}

func setIfPresent(values map[string]any, name string, v *string) {
	if v != nil {
		values[name] = *v
	}
}

// bodySections splits a Markdown body into its "## Title" sections. Kira's
// comment thread lives under its own "## Comments" header, so this naturally
// excludes it from whatever section a schema field names.
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

// ConfigVocab projects datamodel.Config vocab into the enums map Validate
// expects, honoring the same strict/open fallback core.validateItem
// enforces: a vocab is only membership-checked when it (or the labels
// fallback it defers to) is strict.
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
