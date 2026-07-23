package timex

import (
	"fmt"
	"os"
	"time"
)

// Now returns the current time, or the time parsed from KIRA_NOW if set —
// a test-injectable clock for deterministic time-relative output.
func Now() time.Time {
	if env := os.Getenv("KIRA_NOW"); env != "" {
		if t, err := ParseFlexible(env); err == nil {
			return t
		}
	}
	return time.Now()
}

// ParseFlexible parses an RFC3339 timestamp or a bare DateOnly date.
func ParseFlexible(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse(time.DateOnly, s)
}

func CompareRFC3339(a, b string) (cmp int, aOK, bOK bool) {
	ta, ea := time.Parse(time.RFC3339, a)
	tb, eb := time.Parse(time.RFC3339, b)
	aOK, bOK = ea == nil, eb == nil
	if aOK && bOK {
		return ta.Compare(tb), true, true
	}
	return 0, aOK, bOK
}

func Overdue(due string, now time.Time) bool {
	if _, err := time.Parse(time.DateOnly, due); err != nil {
		return false
	}
	return due < now.Format(time.DateOnly)
}

func HumanSince(ts string, now time.Time) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
