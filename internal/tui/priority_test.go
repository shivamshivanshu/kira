package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestPriorityWeightByVocabPosition(t *testing.T) {
	t.Parallel()
	ic := iconSet{mode: datamodel.IconText}

	marks := map[string]int{"P0": 3, "P1": 2, "P2": 1, "P3": 1, "urgent": 0, "": 0}
	for value, want := range marks {
		if got := strings.Count(ic.priorityCell(value), "!"); got != want {
			t.Errorf("priorityCell(%q) has %d marks, want %d (position-derived)", value, got, want)
		}
	}

	for _, mode := range []datamodel.IconMode{datamodel.IconNerd, datamodel.IconEmoji, datamodel.IconText} {
		im := iconSet{mode: mode}
		gutter := 3 * lipgloss.Width(glyphPriority.pick(mode))
		for _, v := range []string{"P0", "P1", "P2", "P3", "urgent", ""} {
			if got := lipgloss.Width(im.priorityCell(v)); got != gutter {
				t.Errorf("%s priorityCell(%q) width = %d, want fixed gutter %d", mode, v, got, gutter)
			}
		}
	}
}

func TestPriorityHueByTier(t *testing.T) {
	t.Parallel()
	th := colorTheme()
	const probe = "x"
	if priorityHue(th, 0).Render(probe) != th.Heat.Hot.Render(probe) {
		t.Error("index 0 must tint with Heat.Hot")
	}
	if priorityHue(th, 1).Render(probe) != th.Heat.Warm.Render(probe) {
		t.Error("index 1 must tint with Heat.Warm")
	}
	for _, tier := range []int{2, 3, -1} {
		if priorityHue(th, tier).Render(probe) != th.Dim.Render(probe) {
			t.Errorf("tier %d must fall back to Dim", tier)
		}
	}
}

func TestTreeRowPriorityMarksFromVocab(t *testing.T) {
	t.Parallel()
	high := "P0"
	nodes := []datamodel.TreeNode{{ID: "T1", Number: "KIRA-1", Type: datamodel.TypeTicket, Title: "Hot"}}
	fields := map[string]datamodel.ListItem{
		"T1": {ID: "T1", Number: "KIRA-1", State: "TODO", Category: "todo", Type: datamodel.TypeTicket, Priority: &high},
	}
	tm := newTreeModel()
	(&tm).load(nodes, fields, map[string]datamodel.ContainerProgress{})
	out := tm.render(asciiTheme(), iconSet{mode: datamodel.IconText}, 100, 3, true, false)
	if !strings.Contains(out, "!!!") {
		t.Errorf("an index-0 priority must render three marks in the row:\n%s", out)
	}
}
