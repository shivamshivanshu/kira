package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func Validate(c *datamodel.Config) error {
	if c.Version != datamodel.SchemaVersion {
		return fmt.Errorf("config: version: unsupported version %d (only %d is known)", c.Version, datamodel.SchemaVersion)
	}
	if err := validateEnum("id.style", c.ID.Style, datamodel.IDStyles...); err != nil {
		return err
	}
	if err := validateEnum("commit.mode", c.Commit.Mode, datamodel.CommitModes...); err != nil {
		return err
	}
	if err := validateEnum("merge.policy", c.Merge.Policy, datamodel.MergePolicies...); err != nil {
		return err
	}
	if err := validateEnum("ui.icons", c.UI.Icons, datamodel.IconModes...); err != nil {
		return err
	}
	if err := validateEnum("ui.background", c.UI.Background, datamodel.Backgrounds...); err != nil {
		return err
	}
	if err := validateEnum("estimate.unit", c.Estimate.Unit, datamodel.EstimateUnits...); err != nil {
		return err
	}
	if err := validateEnum("workon.casing", c.Workon.Casing, datamodel.Casings...); err != nil {
		return err
	}
	if !strings.Contains(c.Workon.BranchPattern, "{number}") {
		return fmt.Errorf("config: workon.branch_pattern: must contain {number}, got %q", c.Workon.BranchPattern)
	}
	if c.Estimate.HoursPerDay <= 0 {
		return fmt.Errorf("config: estimate.hours_per_day: must be > 0, got %v", c.Estimate.HoursPerDay)
	}
	if len(c.Workflows) == 0 {
		return fmt.Errorf("config: workflows: at least one workflow is required")
	}
	for name, wf := range c.Workflows {
		if err := validateWorkflow(name, wf, c); err != nil {
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
	if err := validateFilters(c); err != nil {
		return err
	}
	if err := validateAutomation(c); err != nil {
		return err
	}
	return validateSprints(c)
}

func validateAutomation(c *datamodel.Config) error {
	for i, h := range c.Automation {
		where := fmt.Sprintf("automation[%d]", i)
		if !slices.Contains(datamodel.AutomationEvents, h.On) {
			return fmt.Errorf("config: %s.on: invalid event %q, want one of %v", where, h.On, datamodel.AutomationEvents)
		}
		if strings.TrimSpace(h.Run) == "" {
			return fmt.Errorf("config: %s.run: required", where)
		}
		if _, err := h.TimeoutDuration(); err != nil {
			return fmt.Errorf("config: %s.timeout: invalid duration %q", where, h.Timeout)
		}
	}
	return nil
}

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

func validateFilters(c *datamodel.Config) error {
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

func validateSprints(c *datamodel.Config) error {
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
		if !datamodel.ValidDate(s.Start) {
			return fmt.Errorf("config: sprints[%s].start: invalid RFC3339 date %q", s.Key, s.Start)
		}
		if !datamodel.ValidDate(s.End) {
			return fmt.Errorf("config: sprints[%s].end: invalid RFC3339 date %q", s.Key, s.End)
		}
		if s.Start >= s.End {
			return fmt.Errorf("config: sprints[%s]: start %s is not before end %s", s.Key, s.Start, s.End)
		}
	}
	return nil
}

func validateWorkflow(name string, w datamodel.Workflow, c *datamodel.Config) error {
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
		if !slices.Contains(datamodel.Categories, s.Category) {
			return fmt.Errorf("config: workflows.%s.states[%s].category: invalid value %q, want one of %v", name, s.Key, s.Category, datamodel.Categories)
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
			if err := validateGuards(t, fmt.Sprintf("workflows.%s.transitions.%s", name, from), c); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateGuards(t datamodel.Transition, where string, c *datamodel.Config) error {
	for _, f := range t.Require {
		if !slices.Contains(datamodel.MutableFields, f) {
			return fmt.Errorf("config: %s: require names unknown field %q", where, f)
		}
	}
	for f, v := range t.Set {
		if !slices.Contains(datamodel.MutableFields, f) {
			return fmt.Errorf("config: %s: set names unknown field %q", where, f)
		}
		if vocab, ok := c.VocabFor(f); ok && len(vocab) > 0 && !slices.Contains(vocab, v) {
			return fmt.Errorf("config: %s: set.%s: %q is not in the configured %s vocabulary", where, f, v, f)
		}
	}
	return nil
}

func validateEnum[T ~string](key string, val T, allowed ...T) error {
	if slices.Contains(allowed, val) {
		return nil
	}
	return fmt.Errorf("config: %s: invalid value %q, want one of %v", key, val, allowed)
}
