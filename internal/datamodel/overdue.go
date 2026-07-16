package datamodel

import (
	"time"

	"github.com/shivamshivanshu/kira/internal/timex"
)

// IsOverdue reports whether due names a date in the past. Nil due or a
// done-category item are never overdue.
func IsOverdue(due *string, category string, now time.Time) bool {
	if due == nil || category == string(CategoryDone) {
		return false
	}
	return timex.Overdue(*due, now)
}
