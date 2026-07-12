package config

import (
	"fmt"
	"slices"
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
		if err := wf.validate(name); err != nil {
			return err
		}
	}
	return nil
}

func (w Workflow) validate(name string) error {
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
		for _, to := range targets {
			if !defined[to] {
				return fmt.Errorf("config: workflows.%s.transitions.%s: unknown target state %q", name, from, to)
			}
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
