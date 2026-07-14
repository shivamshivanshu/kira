package tui

import (
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func overdue(due *string, category string) bool {
	if due == nil || *due == "" || datamodel.Category(category) == datamodel.CategoryDone {
		return false
	}
	return *due < time.Now().Format(time.DateOnly)
}
