package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/item"
)

// Validate rejects a config that would break kira's invariants, returning an
// error naming the offending key. It is the gate every loaded config passes.
func (c *Config) Validate() error {
	if c.Version != SchemaVersion {
		return fmt.Errorf("config: version: unsupported version %d (only %d is known)", c.Version, SchemaVersion)
	}
	if err := enum("id.style", c.ID.Style, idStyles...); err != nil {
		return err
	}
	if err := enum("commit.mode", c.Commit.Mode, commitModes...); err != nil {
		return err
	}
	if err := enum("merge.policy", c.Merge.Policy, mergePolicies...); err != nil {
		return err
	}
	if err := enum("ui.icons", c.UI.Icons, iconModes...); err != nil {
		return err
	}
	if err := enum("estimate.unit", c.Estimate.Unit, estimateUnits...); err != nil {
		return err
	}
	if c.Estimate.HoursPerDay <= 0 {
		return fmt.Errorf("config: estimate.hours_per_day: must be > 0, got %v", c.Estimate.HoursPerDay)
	}
	if len(c.Workflows) == 0 {
		return fmt.Errorf("config: workflows: at least one workflow is required")
	}
	for name, wf := range c.Workflows {
		if err := wf.validate(name, c); err != nil {
			return err
		}
	}
	if err := validateVocabList("priorities", c.Priorities); err != nil {
		return err
	}
	if err := validateVocabList("subtypes", c.Subtypes); err != nil {
		return err
	}
	if err := validateVocabList("resolutions", c.Resolutions); err != nil {
		return err
	}
	if err := c.validateFilters(); err != nil {
		return err
	}
	return c.validateSprints()
}

// VocabFor returns the configured vocabulary list governing an enum-ish item
// field (priority/subtype/resolution) and whether the field is vocab-governed
// at all. It is the one home of the field→list mapping, shared with core's
// item validation; an empty list means the field is free-form.
func (c *Config) VocabFor(field string) ([]string, bool) {
	switch field {
	case "priority":
		return c.Priorities, true
	case "subtype":
		return c.Subtypes, true
	case "resolution":
		return c.Resolutions, true
	}
	return nil, false
}

// validateVocabList rejects empty or duplicate entries in an ordered vocabulary
// list (priorities/subtypes/resolutions) — duplicates would break the ranked
// order priorities defines.
func validateVocabList(key string, list []string) error {
	seen := make(map[string]bool, len(list))
	for _, v := range list {
		if v == "" {
			return fmt.Errorf("config: %s: empty entry", key)
		}
		if seen[v] {
			return fmt.Errorf("config: %s: duplicate entry %q", key, v)
		}
		seen[v] = true
	}
	return nil
}

func (c *Config) validateFilters() error {
	for name, query := range c.Filters {
		if name == "" {
			return fmt.Errorf("config: filters: empty filter name")
		}
		if strings.TrimSpace(query) == "" {
			return fmt.Errorf("config: filters.%s: empty query", name)
		}
	}
	return nil
}

func (c *Config) validateSprints() error {
	keys := make(map[string]bool, len(c.Sprints))
	for _, s := range c.Sprints {
		if s.Key == "" {
			return fmt.Errorf("config: sprints: sprint with empty key")
		}
		if keys[s.Key] {
			return fmt.Errorf("config: sprints: duplicate key %q", s.Key)
		}
		keys[s.Key] = true
		if s.Name == "" {
			return fmt.Errorf("config: sprints[%s]: empty name", s.Key)
		}
		if !item.ValidDate(s.Start) {
			return fmt.Errorf("config: sprints[%s].start: invalid RFC3339 date %q", s.Key, s.Start)
		}
		if !item.ValidDate(s.End) {
			return fmt.Errorf("config: sprints[%s].end: invalid RFC3339 date %q", s.Key, s.End)
		}
		if s.Start >= s.End {
			return fmt.Errorf("config: sprints[%s]: start %s is not before end %s", s.Key, s.Start, s.End)
		}
	}
	return nil
}

// Sprint returns the configured sprint named key.
func (c *Config) Sprint(key string) (Sprint, bool) {
	for _, s := range c.Sprints {
		if s.Key == key {
			return s, true
		}
	}
	return Sprint{}, false
}

// HasSprint reports whether key names a configured sprint.
func (c *Config) HasSprint(key string) bool {
	_, ok := c.Sprint(key)
	return ok
}

func (w Workflow) validate(name string, c *Config) error {
	if len(w.States) == 0 {
		return fmt.Errorf("config: workflows.%s.states: workflow has no states", name)
	}
	defined := make(map[string]bool, len(w.States))
	for _, s := range w.States {
		if s.Key == "" {
			return fmt.Errorf("config: workflows.%s.states: state with empty key", name)
		}
		if defined[s.Key] {
			return fmt.Errorf("config: workflows.%s.states: duplicate state %q", name, s.Key)
		}
		defined[s.Key] = true
		if !slices.Contains(categories, s.Category) {
			return fmt.Errorf("config: workflows.%s.states[%s].category: invalid value %q, want one of %v", name, s.Key, s.Category, categories)
		}
		if s.Wip < 0 {
			return fmt.Errorf("config: workflows.%s.states[%s].wip: must be >= 0, got %d", name, s.Key, s.Wip)
		}
	}
	if w.Initial == "" {
		return fmt.Errorf("config: workflows.%s.initial: required", name)
	}
	if !defined[w.Initial] {
		return fmt.Errorf("config: workflows.%s.initial: %q is not a defined state", name, w.Initial)
	}
	for from, targets := range w.Transitions {
		if !defined[from] {
			return fmt.Errorf("config: workflows.%s.transitions: unknown state %q", name, from)
		}
		for _, t := range targets {
			if t.To == "" {
				return fmt.Errorf("config: workflows.%s.transitions.%s: transition without a target state", name, from)
			}
			if !defined[t.To] {
				return fmt.Errorf("config: workflows.%s.transitions.%s: unknown target state %q", name, from, t.To)
			}
			if err := t.validateGuards(fmt.Sprintf("workflows.%s.transitions.%s", name, from), c); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateGuards checks a transition's require:/set: guards name known item
// fields (item.MutableFields — the schema's own list, so a new item field can
// never drift out of guard reach) and that set: values satisfy the field's
// configured vocabulary (docs/design/02-data-model.md §10).
func (t Transition) validateGuards(where string, c *Config) error {
	for _, f := range t.Require {
		if !slices.Contains(item.MutableFields, f) {
			return fmt.Errorf("config: %s: require names unknown field %q", where, f)
		}
	}
	for f, v := range t.Set {
		if !slices.Contains(item.MutableFields, f) {
			return fmt.Errorf("config: %s: set names unknown field %q", where, f)
		}
		if vocab, ok := c.VocabFor(f); ok && len(vocab) > 0 && !slices.Contains(vocab, v) {
			return fmt.Errorf("config: %s: set.%s: %q is not in the configured %s vocabulary", where, f, v, f)
		}
	}
	return nil
}

// enum returns an error naming key when val is not one of allowed. The string
// type parameter keeps each call's val and allowed set the same enum type, so a
// mismatched constant set is a compile error rather than a silent pass.
func enum[T ~string](key string, val T, allowed ...T) error {
	if slices.Contains(allowed, val) {
		return nil
	}
	return fmt.Errorf("config: %s: invalid value %q, want one of %v", key, val, allowed)
}
