package config

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/query"
	"github.com/shivamshivanshu/kira/internal/setx"
)

// BoardKeyPattern is the regex pattern for valid board key identifiers.
const BoardKeyPattern = `^[A-Z][A-Z0-9]{1,9}$`

// MaxBoardKeyLen is the length cap BoardKeyPattern enforces.
const MaxBoardKeyLen = 10

var boardKeyRe = regexp.MustCompile(BoardKeyPattern)

func ValidBoardKey(key string) bool {
	return boardKeyRe.MatchString(key)
}

func Validate(c *datamodel.Config) error {
	if c.Version < 1 || c.Version > datamodel.SchemaVersion {
		return errx.User("config: version: unsupported version %d (this kira understands 1..%d)", c.Version, datamodel.SchemaVersion)
	}
	if err := validateEnum("commit.mode", c.Commit.Mode, datamodel.CommitModes...); err != nil {
		return err
	}
	if err := validateEnum("merge.policy", c.Merge.Policy, datamodel.MergePolicies...); err != nil {
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
	for i, p := range c.People.Known {
		if p.Name == "" {
			return errx.User("config: people.known[%d].name: required", i)
		}
	}
	if err := validateFilters(c); err != nil {
		return err
	}
	if err := validateBoards(c); err != nil {
		return err
	}
	return validateSprints(c)
}

func validateBoards(c *datamodel.Config) error {
	if c.Boards != nil && c.Version < datamodel.BoardsSchemaVersion {
		return errx.User("config: boards require version %d, found %d", datamodel.BoardsSchemaVersion, c.Version).
			WithHint("bump version to %d or remove the boards list", datamodel.BoardsSchemaVersion)
	}
	seen := setx.NewDeduper[string]()
	defaults := 0
	for i, b := range c.Boards {
		where := fmt.Sprintf("boards[%d]", i)
		if !ValidBoardKey(b.Key) {
			return errx.User("config: %s.key: %q must match %s", where, b.Key, BoardKeyPattern)
		}
		up := strings.ToUpper(b.Key)
		if !seen.Add(up) {
			return errx.User("config: boards: duplicate key %q", b.Key)
		}
		if strings.TrimSpace(b.Name) == "" {
			return errx.User("config: %s.name: required", where)
		}
		if b.Default {
			defaults++
		}
	}
	if defaults > 1 {
		return errx.User("config: boards: at most one board may be default, found %d", defaults)
	}
	return nil
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
		return errx.User("config: ui.tui.refresh: invalid duration %q", s)
	}
	if d != 0 && d < datamodel.MinRefreshInterval {
		return errx.User("config: ui.tui.refresh: %s is below the %s minimum", d, datamodel.MinRefreshInterval).
			WithHint("use 0 to disable")
	}
	return nil
}

func validateWorkonSection(w datamodel.Workon) error {
	if !strings.Contains(w.BranchPattern, "{number}") {
		return errx.User("config: workon.branch_pattern: must contain {number}, got %q", w.BranchPattern)
	}
	if strings.HasPrefix(w.WorktreeDir, "~") {
		return errx.User("config: workon.worktree_dir: %q begins with ~, which is not expanded", w.WorktreeDir).
			WithHint("use an absolute or repo-relative path")
	}
	return nil
}

func UIWarnings(ui datamodel.UI) []string {
	var out []string
	// ui.list.columns is deliberately unchecked here: `kira list` warns about
	// and drops unknown columns at render time.
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

func validateFilters(c *datamodel.Config) error {
	for name, q := range c.Filters {
		if name == "" {
			return errx.User("config: filters: empty filter name")
		}
		// Unknown fields pass here: every use site compiles the saved filter
		// and reports the unknown field there; rejecting it at load would
		// brick every command for a repo whose saved filter names one.
		if _, err := query.Parse(q); err != nil && !query.IsUnknownField(err) {
			return errx.User("config: filters.%s: %v", name, err)
		}
	}
	return nil
}

func validateSprints(c *datamodel.Config) error {
	keys := make(map[string]bool, len(c.Sprints))
	for _, s := range c.Sprints {
		if s.Key == "" {
			return errx.User("config: sprints: sprint with empty key")
		}
		if keys[s.Key] {
			return errx.User("config: sprints: duplicate key %q", s.Key)
		}
		keys[s.Key] = true
		if s.Name == "" {
			return errx.User("config: sprints[%s]: empty name", s.Key)
		}
		if !datamodel.ValidDate(s.Start) {
			return errx.User("config: sprints[%s].start: invalid date %q, want YYYY-MM-DD", s.Key, s.Start)
		}
		if !datamodel.ValidDate(s.End) {
			return errx.User("config: sprints[%s].end: invalid date %q, want YYYY-MM-DD", s.Key, s.End)
		}
		if s.Start >= s.End {
			return errx.User("config: sprints[%s]: start %s is not before end %s", s.Key, s.Start, s.End)
		}
	}
	return nil
}

func validateEnum[T ~string](key string, val T, allowed ...T) error {
	if slices.Contains(allowed, val) {
		return nil
	}
	return errx.User("config: %s: invalid value %q, want one of %v", key, val, allowed)
}
