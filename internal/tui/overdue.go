package tui

import (
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/timex"
)

func overdue(due *string, category string) bool {
	if due == nil {
		return false
	}
	return timex.Overdue(*due, datamodel.Category(category) == datamodel.CategoryDone, time.Now())
}
