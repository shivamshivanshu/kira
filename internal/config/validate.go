package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

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
	if strings.TrimSpace(c.Commit.Trailer) == "" {
		return fmt.Errorf("config: commit.trailer: required; an empty trailer key disables all commit linking")
	}
	if strings.ContainsAny(c.Commit.SubjectPrefix, "\n\r") {
		return fmt.Errorf("config: commit.subject_prefix: must be a single line, got %q", c.Commit.SubjectPrefix)
	}
	for i, m := range c.Commit.LinkMarkers {
		if err := validateEnum(fmt.Sprintf("commit.link_markers[%d]", i), m, datamodel.LinkMarkers...); err != nil {
			return err
		}
	}
	for i, m := range c.Commit.ReferenceMarkers {
		if err := validateEnum(fmt.Sprintf("commit.reference_markers[%d]", i), m, datamodel.ReferenceMarkers...); err != nil {
			return err
		}
	}
	if err := validateEnum("merge.policy", c.Merge.Policy, datamodel.MergePolicies...); err != nil {
		return err
	}
	if err := validateEnum("sync.dirty", c.Sync.Dirty, datamodel.SyncDirties...); err != nil {
		return err
	}
	if err := validateUISection(c.UI); err != nil {
		return err
	}
	if err := validateEnum("estimate.unit", c.Estimate.Unit, datamodel.EstimateUnits...); err != nil {
		return err
	}
	if err := validateWorkonSection(c.Workon); err != nil {
		return err
	}
	if len(c.Workflows) == 0 {
		return fmt.Errorf("config: workflows: at least one workflow is required")
	}
	for name, wf := range c.Workflows {
		if err := validateWorkflow(name, wf, c); err != nil {
			return err
		}
	}
	for _, vl := range []struct {
		key  string
		list []string
	}{
		{"priorities", c.Priorities.Values},
		{"subtypes", c.Subtypes.Values},
		{"resolutions", c.Resolutions.Values},
		{"resolutions_dropped", c.ResolutionsDropped},
	} {
		if err := validateVocabList(vl.key, vl.list); err != nil {
			return err
		}
	}
	if len(c.Resolutions.Values) > 0 {
		for _, d := range c.ResolutionsDropped {
			if !slices.Contains(c.Resolutions.Values, d) {
				return fmt.Errorf("config: resolutions_dropped: %q is not one of the configured resolutions %v; dropped detection would never match", d, c.Resolutions.Values)
			}
		}
	}
	for i, p := range c.People.Known {
		if p.Name == "" {
			return fmt.Errorf("config: people.known[%d].name: required", i)
		}
	}
	if err := validateFilters(c); err != nil {
		return err
	}
	if err := validateAutomation(c); err != nil {
		return err
	}
	return validateSprints(c)
}

func validateUISection(ui datamodel.UI) error {
	if err := validateEnum("ui.icons", ui.Icons, datamodel.IconModes...); err != nil {
		return err
	}
	if err := validateEnum("ui.background", ui.Background, datamodel.Backgrounds...); err != nil {
		return err
	}
	if err := validateEnum("ui.color", ui.Color, datamodel.ColorModes...); err != nil {
		return err
	}
	return validateRefresh(ui.Tui.Refresh)
}

func validateRefresh(s string) error {
	d, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("config: ui.tui.refresh: invalid duration %q", s)
	}
	if d != 0 && d < datamodel.MinRefreshInterval {
		return fmt.Errorf("config: ui.tui.refresh: %s is below the %s minimum (use 0 to disable)", d, datamodel.MinRefreshInterval)
	}
	return nil
}

func validateWorkonSection(w datamodel.Workon) error {
	if err := validateEnum("workon.casing", w.Casing, datamodel.Casings...); err != nil {
		return err
	}
	if !strings.Contains(w.BranchPattern, "{number}") {
		return fmt.Errorf("config: workon.branch_pattern: must contain {number}, got %q", w.BranchPattern)
	}
	if strings.HasPrefix(w.WorktreeDir, "~") {
		return fmt.Errorf("config: workon.worktree_dir: %q begins with ~, which is not expanded; use an absolute or repo-relative path", w.WorktreeDir)
	}
	return nil
}

func UIWarnings(ui datamodel.UI) []string {
	var out []string
	for _, c := range ui.List.Columns {
		if !slices.Contains(datamodel.ListColumns, c) {
			out = append(out, fmt.Sprintf("ui.list.columns: unknown column %q", c))
		}
	}
	for _, slot := range slices.Sorted(maps.Keys(ui.Theme)) {
		switch {
		case !slices.Contains(datamodel.ThemeSlots, slot):
			out = append(out, fmt.Sprintf("ui.theme: unknown slot %q", slot))
		case !datamodel.IsHexColor(ui.Theme[slot]):
			out = append(out, fmt.Sprintf("ui.theme.%s: invalid color %q", slot, ui.Theme[slot]))
		}
	}
	if s := ui.Tui.Split; s <= 0 || s >= 1 {
		out = append(out, fmt.Sprintf("ui.tui.split: %v out of range (0,1)", s))
	}
	return out
}

func validateAutomation(c *datamodel.Config) error {
	return validateAutomationHooks("automation", c.Automation)
}

func validateAutomationHooks(where string, hooks []datamodel.AutomationHook) error {
	for i, h := range hooks {
		at := fmt.Sprintf("%s[%d]", where, i)
		if !slices.Contains(datamodel.AutomationEvents, h.On) {
			return fmt.Errorf("config: %s.on: invalid event %q, want one of %v", at, h.On, datamodel.AutomationEvents)
		}
		if strings.TrimSpace(h.Run) == "" {
			return fmt.Errorf("config: %s.run: required", at)
		}
		if _, err := h.TimeoutDuration(); err != nil {
			return fmt.Errorf("config: %s.timeout: invalid duration %q", at, h.Timeout)
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
	if w.WipPolicy != "" {
		if err := validateEnum(fmt.Sprintf("workflows.%s.wip_policy", name), w.WipPolicy, datamodel.WipPolicies...); err != nil {
			return err
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
		if f == datamodel.RequireBlockersClosed {
			continue
		}
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
