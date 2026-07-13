package core

import (
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
	li := listItemOf(&datamodel.Config{}, it)
	if li.Priority == nil || *li.Priority != priority {
		t.Errorf("priority not carried into ListItem: %v", li.Priority)
	}
	if li.Resolution == nil || *li.Resolution != resolution {
		t.Errorf("resolution not carried into ListItem: %v", li.Resolution)
	}
}
