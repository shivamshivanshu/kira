package core

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/ptr"
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
	cfg := config.Default()
	items := []*datamodel.Item{
		mkItem("KIRA-1", "01A", nil, nil),
		mkItem("KIRA-2", "01B", ptr.To("mm"), nil),
		mkItem("KIRA-3", "01C", nil, ptr.To("P2")),
		mkItem("KIRA-4", "01D", ptr.To("aa"), ptr.To("P3")),
		mkItem("KIRA-5", "01E", nil, ptr.To("P0")),
		mkItem("KIRA-6", "01F", nil, ptr.To("P2")),
	}
	sortByPrecedence(cfg, items)
	want := "KIRA-4,KIRA-2,KIRA-5,KIRA-3,KIRA-6,KIRA-1"
	if got := numbers(items); got != want {
		t.Errorf("precedence order = %s, want %s", got, want)
	}
}

func TestSortByPrecedenceLegacyDegradation(t *testing.T) {
	cfg := config.Default()
	cfg.Priorities = datamodel.EnumVocab{}
	items := []*datamodel.Item{
		mkItem("KIRA-3", "01C", nil, nil),
		mkItem("KIRA-1", "01A", nil, nil),
		mkItem("HASH-x", "01Z", nil, nil),
		mkItem("KIRA-2", "01B", nil, nil),
	}
	sortByPrecedence(cfg, items)
	legacy := []*datamodel.Item{items[0], items[1], items[2], items[3]}
	sortByKey(legacy, func(it *datamodel.Item) id.SortKey { return id.NewSortKey(it.Number, it.ID) })
	if got, want := numbers(items), numbers(legacy); got != want {
		t.Errorf("degraded order = %s, want legacy %s", got, want)
	}
	items[0].Priority = ptr.To("high")
	sortByPrecedence(cfg, items)
	if got, want := numbers(items), numbers(legacy); got != want {
		t.Errorf("free-form priority perturbed order: %s, want %s", got, want)
	}
}
