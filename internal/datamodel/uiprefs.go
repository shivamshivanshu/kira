package datamodel

import "time"

const (
	DefaultSplit       = 0.5
	DefaultRefresh     = "0"
	DefaultWorktreeDir = "../{repo}-{branch}"

	MinRefreshInterval = time.Second
)

func (u UITui) RefreshInterval() time.Duration {
	d, err := time.ParseDuration(u.Refresh)
	if err != nil || d < MinRefreshInterval {
		return 0
	}
	return d
}

var ListColumns = []string{
	"number", "title", "type", "state", "category",
	"priority", "owner", "labels", "epic", "epic_number",
	"resolution", "due", "id",
}

var DefaultListColumns = []string{"number", "state", "type", "priority", "title"}

var ThemeSlots = []string{"accent", "dim", "todo", "doing", "done", "warm", "hot"}

func IsHexColor(s string) bool {
	if len(s) != 4 && len(s) != 7 {
		return false
	}
	if s[0] != '#' {
		return false
	}
	for _, r := range s[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
