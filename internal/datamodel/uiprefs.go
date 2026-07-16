package datamodel

import "time"

// Default TUI preference values, used when config doesn't set them explicitly.
const (
	DefaultSplit       = 0.5
	DefaultRefresh     = "0"
	DefaultWorktreeDir = "../{repo}-{branch}"

	MinRefreshInterval = time.Second
)

// RefreshInterval parses u.Refresh, returning 0 (no auto-refresh) if it's
// unset, invalid, or below MinRefreshInterval.
func (u UITui) RefreshInterval() time.Duration {
	d, err := time.ParseDuration(u.Refresh)
	if err != nil || d < MinRefreshInterval {
		return 0
	}
	return d
}

// ListColumns are the columns selectable for the list view, in canonical order.
var ListColumns = []string{
	"number", "title", "type", "state", "category",
	"priority", "owner", "labels", "epic", "epic_number",
	"resolution", "due", "id",
}

// DefaultListColumns are the columns shown when config doesn't customize them.
var DefaultListColumns = []string{"number", "state", "type", "priority", "title"}

// ThemeSlots are the named color slots a theme must fill.
var ThemeSlots = []string{"accent", "dim", "todo", "doing", "done", "warm", "hot", "border"}

// IsHexColor reports whether s is a "#RGB" or "#RRGGBB" hex color string.
func IsHexColor(s string) bool {
	if len(s) != 4 && len(s) != 7 {
		return false
	}
	if s[0] != '#' {
		return false
	}
	for _, r := range s[1:] {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}
