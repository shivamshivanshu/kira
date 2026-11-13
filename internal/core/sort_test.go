package core

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func mkItem(number, ulid string, rank, priority *string) *datamodel.Item {
	return &datamodel.Item{ID: ulid, Number: number, Type: datamodel.TypeTicket, Rank: rank, Priority: priority}
}

func numbers(items []*datamodel.Item) string {
	nums := make([]string, len(items))
	for i, it := range items {
		nums[i] = it.Number
	}
	return strings.Join(nums, ",")
}

func TestSortByPrecedence(t *testing.T) {
	items := []*datamodel.Item{
		mkItem("KIRA-1", "01A", nil, nil),
		mkItem("KIRA-2", "01B", new("mm"), nil),
		mkItem("KIRA-3", "01C", nil, new("P2")),
		mkItem("KIRA-4", "01D", new("aa"), new("P3")),
		mkItem("KIRA-5", "01E", nil, new("P0")),
		mkItem("KIRA-6", "01F", nil, new("P2")),
	}
	sortByPrecedence(items)
	want := "KIRA-4,KIRA-2,KIRA-5,KIRA-3,KIRA-6,KIRA-1"
	if got := numbers(items); got != want {
		t.Errorf("precedence order = %s, want %s", got, want)
	}
}
