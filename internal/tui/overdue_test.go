package tui

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func treeWithDue(state, category, due string) treeModel {
	nodes := []datamodel.TreeNode{{ID: "T1", Number: "KIRA-1", Type: datamodel.TypeTicket, Title: "Late"}}
	fields := map[string]datamodel.ListItem{
		"T1": {ID: "T1", Number: "KIRA-1", State: state, Category: category, Type: datamodel.TypeTicket, Due: &due},
	}
	tm := newTreeModel()
	(&tm).load(nodes, fields, map[string]datamodel.EpicProgress{})
	return tm
}

func TestDetailPanelRendersDueAndOverdue(t *testing.T) {
	past, future := "2020-01-01", "2099-01-01"
	th := asciiTheme()

	res := sampleDetail()
	res.Due = &future
	if got := newDetailPanel().render(th, iconSet{mode: datamodel.IconText}, res, 100, 40); !strings.Contains(got, "due "+future) {
		t.Errorf("detail must render the due date:\n%s", got)
	} else if strings.Contains(got, "overdue") {
		t.Errorf("a future due date must not be flagged overdue:\n%s", got)
	}

	res.Due = &past
	if got := newDetailPanel().render(th, iconSet{mode: datamodel.IconText}, res, 100, 40); !strings.Contains(got, "overdue") {
		t.Errorf("a past-due active item must be flagged overdue:\n%s", got)
	}

	res.Category = "done"
	if got := newDetailPanel().render(th, iconSet{mode: datamodel.IconText}, res, 100, 40); strings.Contains(got, "overdue") {
		t.Errorf("a done item must never be flagged overdue:\n%s", got)
	}

	res.Due = nil
	res.Category = "doing"
	if got := newDetailPanel().render(th, iconSet{mode: datamodel.IconText}, res, 100, 40); strings.Contains(got, "due ") {
		t.Errorf("an item without a due date must not render a due row:\n%s", got)
	}
}

func TestTreeRowOverdueSignal(t *testing.T) {
	th := asciiTheme()
	render := func(state, cat, due string, mode datamodel.IconMode) string {
		tm := treeWithDue(state, cat, due)
		return tm.render(th, iconSet{mode: mode}, 100, 3, true, false)
	}

	if out := render("TODO", "todo", "2020-01-01", datamodel.IconText); !strings.Contains(out, "!") {
		t.Errorf("overdue active row must show the ascii overdue marker:\n%s", out)
	}
	if out := render("TODO", "todo", "2099-01-01", datamodel.IconText); strings.Contains(out, "!") {
		t.Errorf("a future due date must not mark the row overdue:\n%s", out)
	}
	if out := render("DONE", "done", "2020-01-01", datamodel.IconText); strings.Contains(out, "!") {
		t.Errorf("a done row must never be marked overdue:\n%s", out)
	}
	if out := render("TODO", "todo", "2020-01-01", datamodel.IconNerd); !strings.Contains(out, glyphOverdue.nerd) || strings.Contains(out, "!") {
		t.Errorf("nerd mode must render the overdue glyph, not the ascii marker:\n%s", out)
	}
}

func TestBoardCardOverdueSignal(t *testing.T) {
	card := func(due string) *datamodel.BoardResult {
		d := due
		return &datamodel.BoardResult{Type: datamodel.TypeTicket, Columns: []datamodel.BoardColumn{
			{State: "TODO", Category: "todo", Count: 1, Items: []datamodel.ListItem{
				{ID: "t1", Number: "KIRA-1", Title: "Late", Type: datamodel.TypeTicket, State: "TODO", Category: "todo", Due: &d},
			}},
		}}
	}
	ic := iconSet{mode: datamodel.IconText}

	if out := renderBoard(asciiTheme(), ic, card("2020-01-01"), 40, 4, -1, -1); !strings.Contains(out, "!") {
		t.Errorf("overdue card must show the ascii overdue marker:\n%s", out)
	}
	if out := renderBoard(asciiTheme(), ic, card("2099-01-01"), 40, 4, -1, -1); strings.Contains(out, "!") {
		t.Errorf("a future due date must not mark the card overdue:\n%s", out)
	}
}
