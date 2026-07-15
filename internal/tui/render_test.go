package tui

import (
	"io"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/muesli/termenv"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/ptr"
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
		"T1": {ID: "T1", Number: "KIRA-140", State: "TODO", Category: "todo", Type: datamodel.TypeTicket, Epic: ptr.To("E1")},
		"T2": {ID: "T2", Number: "KIRA-142", State: "IN_PROGRESS", Category: "doing", Type: datamodel.TypeTicket, Epic: ptr.To("E1")},
	}
	progress := map[string]datamodel.EpicProgress{"E1": {Done: 1, Total: 2}}
	return nodes, fields, progress
}

func loadedTree() treeModel {
	nodes, fields, progress := sampleTree()
	tm := newTreeModel()
	(&tm).load(nodes, fields, progress)
	return tm
}

func newTestModel(w, h int, withData bool) model {
	cfg := &datamodel.Config{UI: datamodel.UI{Tui: datamodel.UITui{Split: datamodel.DefaultSplit}}}
	m := newModel(nil, cfg, asciiTheme(), iconSet{mode: datamodel.IconText}, false)
	m.width, m.height = w, h
	if withData {
		nodes, fields, progress := sampleTree()
		m.screens[viewTree].(*treeScreen).setData(&m, treeData{nodes: nodes, fields: fields, progress: progress})
	}
	return m
}

func TestViewEmptyState(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, false)
	golden.RequireEqual(t, []byte(m.View()))
}

func TestViewTreeSplit(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	golden.RequireEqual(t, []byte(m.View()))
}

func TestViewMediumTier(t *testing.T) {
	t.Parallel()
	m := newTestModel(60, 12, true)
	if splitDetail(m.width) {
		t.Fatalf("width %d should be below MinWidth %d", m.width, MinWidth)
	}
	golden.RequireEqual(t, []byte(m.View()))
}

func TestCollapseViaKey(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	updated, _ := m.Update(key("h"))
	golden.RequireEqual(t, []byte(updated.(model).View()))
}

func TestIconsAsciiVsNerd(t *testing.T) {
	t.Parallel()
	th := asciiTheme()
	tm := loadedTree()

	ascii := tm.render(th, iconSet{mode: datamodel.IconText}, 100, 6, true, true)
	if !strings.Contains(ascii, "[E]") || strings.Contains(ascii, glyphEpic.nerd) {
		t.Errorf("ascii render must use [E], not the PUA glyph:\n%s", ascii)
	}
	nerd := tm.render(th, iconSet{mode: datamodel.IconNerd}, 100, 6, true, true)
	if !strings.Contains(nerd, glyphEpic.nerd) || strings.Contains(nerd, "[E]") {
		t.Errorf("nerd render must use the PUA glyph, not [E]:\n%s", nerd)
	}
	emoji := tm.render(th, iconSet{mode: datamodel.IconEmoji}, 100, 6, true, true)
	if !strings.Contains(emoji, glyphEpic.emoji) || strings.Contains(emoji, "[E]") {
		t.Errorf("emoji render must use the emoji glyph, not [E]:\n%s", emoji)
	}
}

func TestTreeNerdIconsSnapshot(t *testing.T) {
	t.Parallel()
	tm := loadedTree()
	golden.RequireEqual(t, []byte(tm.render(asciiTheme(), iconSet{mode: datamodel.IconNerd}, 100, 6, true, true)))
}

func TestTreeEmojiIconsSnapshot(t *testing.T) {
	t.Parallel()
	tm := loadedTree()
	golden.RequireEqual(t, []byte(tm.render(asciiTheme(), iconSet{mode: datamodel.IconEmoji}, 100, 6, true, true)))
}

func TestIconWidthsUniformPerMode(t *testing.T) {
	t.Parallel()
	groups := map[string][]glyph{
		"type":     {glyphEpic, glyphTicket},
		"category": {glyphTodo, glyphDoing, glyphDone, glyphDropped},
	}
	cellWidth := map[datamodel.IconMode]int{
		datamodel.IconNerd:  1,
		datamodel.IconEmoji: 2,
	}
	for _, mode := range []datamodel.IconMode{datamodel.IconNerd, datamodel.IconEmoji, datamodel.IconText} {
		for name, gs := range groups {
			want := lipgloss.Width(gs[0].pick(mode))
			for _, g := range gs[1:] {
				if got := lipgloss.Width(g.pick(mode)); got != want {
					t.Errorf("%s glyphs misaligned (%s): width %d != %d for %q", name, mode, got, want, g.pick(mode))
				}
			}
			if exp, ok := cellWidth[mode]; ok && want != exp {
				t.Errorf("%s %s glyphs must be %d-cell, got width %d", name, mode, exp, want)
			}
		}
	}
}

func TestCategoryGlyphDroppedVsDone(t *testing.T) {
	t.Parallel()
	dropped := datamodel.ResolutionDropped
	other := "fixed"
	for _, mode := range []datamodel.IconMode{datamodel.IconNerd, datamodel.IconEmoji, datamodel.IconText} {
		ic := iconSet{mode: mode, dropped: []string{dropped}}
		if got := ic.categoryGlyph(datamodel.CategoryDone, &dropped); got != glyphDropped.pick(mode) {
			t.Errorf("%s done+dropped must render dropped glyph, got %q", mode, got)
		}
		if got := ic.categoryGlyph(datamodel.CategoryDone, &other); got != glyphDone.pick(mode) {
			t.Errorf("%s done+non-dropped resolution must render done glyph, got %q", mode, got)
		}
		if got := ic.categoryGlyph(datamodel.CategoryDone, nil); got != glyphDone.pick(mode) {
			t.Errorf("%s done+nil resolution must render done glyph, got %q", mode, got)
		}
	}
}

func TestTreeRowPlumbsPriorityAndResolution(t *testing.T) {
	t.Parallel()
	p0 := "P0"
	dropped := datamodel.ResolutionDropped
	nodes := []datamodel.TreeNode{{ID: "T1", Number: "KIRA-1", Type: datamodel.TypeTicket, Title: "Dropped work"}}
	fields := map[string]datamodel.ListItem{
		"T1": {ID: "T1", Number: "KIRA-1", State: "WONT_DO", Category: "done", Type: datamodel.TypeTicket, Priority: &p0, Resolution: &dropped},
	}
	tm := newTreeModel()
	(&tm).load(nodes, fields, map[string]datamodel.EpicProgress{})
	out := tm.render(asciiTheme(), iconSet{mode: datamodel.IconText, priorities: []string{"P0", "P1", "P2", "P3"}, dropped: []string{dropped}}, 100, 3, true, false)
	if !strings.Contains(out, "!") {
		t.Errorf("P0 priority marker missing from row:\n%s", out)
	}
	if !strings.Contains(out, "[-]") {
		t.Errorf("dropped glyph missing from row:\n%s", out)
	}
	if strings.Contains(out, "[x]") {
		t.Errorf("dropped ticket must not render the done glyph:\n%s", out)
	}
}

func key(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestDetailMemoServesCacheWithoutStore(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	ts := m.screens[viewTree].(*treeScreen)
	cached := &datamodel.ShowResult{ID: "E1", Title: "cached"}
	ts.host.cache["E1"] = cached

	ts.tree.cursor = 0
	ts.syncDetail(&m)
	if ts.host.detail != cached {
		t.Fatalf("syncDetail did not serve the cache (store is nil, so a disk hit would have yielded nil): %+v", ts.host.detail)
	}

	ts.setData(&m, treeData{nodes: ts.tree.nodes, fields: ts.tree.fields, progress: ts.tree.progress})
	if _, ok := ts.host.cache["E1"]; ok {
		t.Fatal("setData did not invalidate the detail memo on reload")
	}
}
