package datamodel

import (
	"testing"
	"time"
)

func TestIsOverdue(t *testing.T) {
	now, err := time.Parse(time.DateOnly, "2026-07-16")
	if err != nil {
		t.Fatal(err)
	}
	past := "2026-07-15"
	future := "2026-07-17"

	if IsOverdue(nil, "todo", now) {
		t.Error("a nil due date must never be overdue")
	}
	if !IsOverdue(&past, "todo", now) {
		t.Error("a past due date on a non-done item must be overdue")
	}
	if IsOverdue(&future, "todo", now) {
		t.Error("a future due date must not be overdue")
	}
	if IsOverdue(&past, string(CategoryDone), now) {
		t.Error("a done-category item must never be overdue, even with a past due date")
	}
}
