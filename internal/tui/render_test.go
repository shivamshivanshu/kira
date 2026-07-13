package tui

import (
	"io"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

func asciiTheme() theme.Theme {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.Ascii)
	r.SetHasDarkBackground(true)
	return theme.New(r)
}

func sampleTree() ([]datamodel.TreeNode, map[string]datamodel.ListItem, map[string]datamodel.EpicProgress) {
	nodes := []datamodel.TreeNode{{
		ID: "E1", Number: "KIRA-100", Type: datamodel.TypeEpic, Title: "Order book hardening",
		Children: []datamodel.TreeNode{
			{ID: "T1", Number: "KIRA-140", Type: datamodel.TypeTicket, Title: "Fix snapshot dedup"},
			{ID: "T2", Number: "KIRA-142", Type: datamodel.TypeTicket, Title: "Fix race in order-book snapshot merge"},
		},
	}}
	fields := map[string]datamodel.ListItem{
		"E1": {ID: "E1", Number: "KIRA-100", State: "OPEN", Category: "doing", Type: datamodel.TypeEpic},
		"T1": {ID: "T1", Number: "KIRA-140", State: "TODO", Category: "todo", Type: datamodel.TypeTicket, Epic: strptr("E1")},
		"T2": {ID: "T2", Number: "KIRA-142", State: "IN_PROGRESS", Category: "doing", Type: datamodel.TypeTicket, Epic: strptr("E1")},
	}
	progress := map[string]datamodel.EpicProgress{"E1": {Done: 1, Total: 2}}
	return nodes, fields, progress
}

func strptr(s string) *string { return &s }

func newTestModel(w, h int, withData bool) model {
	m := newModel(nil, nil, asciiTheme(), iconSet{nerd: false}, false)
	m.width, m.height = w, h
	if withData {
		nodes, fields, progress := sampleTree()
		m.screens[viewTree].(*treeScreen).apply(&m, treeData{nodes: nodes, fields: fields, progress: progress})
	}
	return m
}

func TestViewEmptyState(t *testing.T) {
	m := newTestModel(100, 12, false)
	golden.RequireEqual(t, []byte(m.View()))
}

func TestViewTreeSplit(t *testing.T) {
	m := newTestModel(100, 12, true)
	golden.RequireEqual(t, []byte(m.View()))
}

func TestViewMediumTier(t *testing.T) {
	m := newTestModel(60, 12, true)
	if splitDetail(m.width) {
		t.Fatalf("width %d should be below MinWidth %d", m.width, MinWidth)
	}
	golden.RequireEqual(t, []byte(m.View()))
}

func TestCollapseViaKey(t *testing.T) {
	m := newTestModel(100, 12, true)
	updated, _ := m.Update(key("h"))
	golden.RequireEqual(t, []byte(updated.(model).View()))
}

func TestTeatestCollapseSnapshot(t *testing.T) {
	m := newTestModel(100, 12, true)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 12))
	tm.Type("h")
	tm.Type("q")
	tm.WaitFinished(t)
	golden.RequireEqual(t, []byte(tm.FinalModel(t).(model).View()))
}

func TestIconsAsciiVsNerd(t *testing.T) {
	nodes, fields, progress := sampleTree()
	th := asciiTheme()
	tm := newTreeModel()
	(&tm).load(nodes, fields, progress)

	ascii := tm.render(th, iconSet{nerd: false}, 100, 6, true, true)
	if !strings.Contains(ascii, "[E]") || strings.Contains(ascii, glyphEpic.nerd) {
		t.Errorf("ascii render must use [E], not the PUA glyph:\n%s", ascii)
	}
	nerd := tm.render(th, iconSet{nerd: true}, 100, 6, true, true)
	if !strings.Contains(nerd, glyphEpic.nerd) || strings.Contains(nerd, "[E]") {
		t.Errorf("nerd render must use the PUA glyph, not [E]:\n%s", nerd)
	}
}

func TestTreeNerdIconsSnapshot(t *testing.T) {
	nodes, fields, progress := sampleTree()
	tm := newTreeModel()
	(&tm).load(nodes, fields, progress)
	golden.RequireEqual(t, []byte(tm.render(asciiTheme(), iconSet{nerd: true}, 100, 6, true, true)))
}

func TestPriorityGlyph(t *testing.T) {
	nerd := iconSet{nerd: true}
	ascii := iconSet{nerd: false}
	for _, p := range []string{"P0", "P1"} {
		if nerd.priorityGlyph(p) != glyphPriority.nerd {
			t.Errorf("nerd priority glyph for %s = %q, want the PUA marker", p, nerd.priorityGlyph(p))
		}
		if ascii.priorityGlyph(p) != "!" {
			t.Errorf("ascii priority glyph for %s = %q, want !", p, ascii.priorityGlyph(p))
		}
	}
	for _, p := range []string{"P2", "P3", ""} {
		if got := nerd.priorityGlyph(p); got != "" {
			t.Errorf("priority %q must have no marker, got %q", p, got)
		}
	}
}

func TestIconWidthsUniformPerMode(t *testing.T) {
	groups := map[string][]glyph{
		"type":     {glyphEpic, glyphTicket},
		"category": {glyphTodo, glyphDoing, glyphDone, glyphDropped},
	}
	for _, nerd := range []bool{false, true} {
		for name, gs := range groups {
			want := lipgloss.Width(gs[0].pick(nerd))
			for _, g := range gs[1:] {
				if got := lipgloss.Width(g.pick(nerd)); got != want {
					t.Errorf("%s glyphs misaligned (nerd=%v): width %d != %d for %q", name, nerd, got, want, g.pick(nerd))
				}
			}
			if nerd && want != 1 {
				t.Errorf("%s PUA glyphs must be single-cell, got width %d", name, want)
			}
		}
	}
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestDetailMemoServesCacheWithoutStore(t *testing.T) {
	m := newTestModel(100, 12, true)
	ts := m.screens[viewTree].(*treeScreen)
	cached := &datamodel.ShowResult{ID: "E1", Title: "cached"}
	ts.detailCache["E1"] = cached

	ts.tree.cursor = 0
	ts.syncDetail(&m)
	if ts.detail != cached {
		t.Fatalf("syncDetail did not serve the cache (store is nil, so a disk hit would have yielded nil): %+v", ts.detail)
	}

	ts.apply(&m, treeData{nodes: ts.tree.nodes, fields: ts.tree.fields, progress: ts.tree.progress})
	if _, ok := ts.detailCache["E1"]; ok {
		t.Fatal("apply did not invalidate the detail memo on reload")
	}
}
