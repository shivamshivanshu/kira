package core

import (
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestListItemOfCarriesPriorityAndResolution(t *testing.T) {
	priority := "P1"
	resolution := datamodel.ResolutionDropped
	it := &datamodel.Item{
		ID: "i1", Number: "KIRA-1", Type: datamodel.TypeTicket, State: "WONT_DO",
		Priority: &priority, Resolution: &resolution,
	}
	li := listItemOf(it, nil)
	if li.Priority == nil || *li.Priority != priority {
		t.Errorf("priority not carried into ListItem: %v", li.Priority)
	}
	if li.Resolution == nil || *li.Resolution != resolution {
		t.Errorf("resolution not carried into ListItem: %v", li.Resolution)
	}
}

func TestGroupByParentNestsContainers(t *testing.T) {
	rootEpic := "epic-root"
	nestedEpic := "epic-nested"
	rows := []datamodel.ListItem{
		{ID: rootEpic, Number: "KIRA-1", Type: datamodel.TypeEpic},
		{ID: nestedEpic, Number: "KIRA-2", Type: datamodel.TypeEpic, Parent: &rootEpic},
		{ID: "ticket", Number: "KIRA-3", Type: datamodel.TypeTicket, Parent: &nestedEpic},
		{ID: "orphan", Number: "KIRA-4", Type: datamodel.TypeTicket},
	}
	parentNumbers := map[string]string{rootEpic: "KIRA-1", nestedEpic: "KIRA-2"}
	groups := groupByParent(rows, parentNumbers)

	items := map[string][]string{}
	for _, g := range groups {
		key := ""
		if g.Parent != nil {
			key = *g.Parent
		}
		items[key] = g.Items
	}
	if len(items) != 3 {
		t.Fatalf("expected root, nested and orphan groups, got %+v", groups)
	}
	if want := []string{nestedEpic}; !slices.Equal(items[rootEpic], want) {
		t.Errorf("root epic group = %v, want %v", items[rootEpic], want)
	}
	if want := []string{"ticket"}; !slices.Equal(items[nestedEpic], want) {
		t.Errorf("nested epic group = %v, want %v", items[nestedEpic], want)
	}
	if want := []string{"orphan"}; !slices.Equal(items[""], want) {
		t.Errorf("orphan group = %v, want %v", items[""], want)
	}
}

func TestGroupByParentKeepsRootContainersOutOfTheOrphanGroup(t *testing.T) {
	rows := []datamodel.ListItem{
		{ID: "epic", Number: "KIRA-1", Type: datamodel.TypeEpic},
		{ID: "ticket", Number: "KIRA-2", Type: datamodel.TypeTicket},
	}
	groups := groupByParent(rows, map[string]string{"epic": "KIRA-1"})
	for _, g := range groups {
		if g.Parent == nil && !slices.Equal(g.Items, []string{"ticket"}) {
			t.Fatalf("orphan group = %v, want [ticket]", g.Items)
		}
	}
}
