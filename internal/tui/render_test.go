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
	"github.com/shivamshivanshu/kira/internal/entityschema"
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
		"T1": {ID: "T1", Number: "KIRA-140", State: "TODO", Category: "todo", Type: datamodel.TypeTicket, Epic: new("E1")},
		"T2": {ID: "T2", Number: "KIRA-142", State: "IN_PROGRESS", Category: "doing", Type: datamodel.TypeTicket, Epic: new("E1")},
	}
	progress := map[string]datamodel.EpicProgress{"E1": {Done: 1, Total: 2}}
	return nodes, fields, progress
}

// declaredIcons supplies glyphs of deliberately differing display widths -- 1
// cell for nerd, 2 for emoji, 3 for the bracketed text form -- so the layout
// code is exercised at each. It is NOT what the shipped schemas declare: those
// declare one mode-independent glyph, which
// TestShippedSchemaIconsIgnoreIconMode pins separately.
func declaredIcons(mode datamodel.IconMode) iconSet {
	glyphs := map[datamodel.IconMode]map[string]string{
		datamodel.IconNerd:  {datamodel.TypeEpic: "\uf45d", datamodel.TypeTicket: "\uf41b"},
		datamodel.IconEmoji: {datamodel.TypeEpic: "\U0001F4E6", datamodel.TypeTicket: "\U0001F3AB"},
	}
	icons, ok := glyphs[mode]
	if !ok {
		icons = map[string]string{datamodel.TypeEpic: "[E]", datamodel.TypeTicket: "[T]"}
	}
	return iconSet{mode: mode}.withTypeIcons(icons)
}

func loadedTree() treeModel {
	nodes, fields, progress := sampleTree()
	tm := newTreeModel()
	(&tm).load(nodes, fields, progress)
	return tm
}

func newTestModel(w, h int, withData bool) model {
	cfg := &datamodel.Config{UI: datamodel.UI{Tui: datamodel.UITui{Split: datamodel.DefaultSplit}}, Workon: datamodel.Workon{BranchPattern: "{key}/{number}-{slug}", Casing: datamodel.CasingKebab}}
	m := newModel(nil, cfg, asciiTheme(), declaredIcons(datamodel.IconText), false)
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

func TestTypeGlyphComesFromTheSchema(t *testing.T) {
	t.Parallel()
	th := asciiTheme()
	tm := loadedTree()

	// kira ships no per-type glyph of its own, so a tree rendered without
	// schema icons draws none -- it does not fall back to a hardcoded one.
	bare := tm.render(th, iconSet{mode: datamodel.IconNerd}, 100, 6, true, true)
	if strings.ContainsAny(bare, "\uf41b\uf45d") {
		t.Errorf("no schema icon was declared, so no type glyph should appear:\n%s", bare)
	}

	// what a schema declares is what gets drawn, verbatim
	declared := iconSet{mode: datamodel.IconNerd}.withTypeIcons(map[string]string{
		datamodel.TypeEpic: "\uf45d", datamodel.TypeTicket: "@",
	})
	out := tm.render(th, declared, 100, 6, true, true)
	if !strings.Contains(out, "\uf45d") {
		t.Errorf("declared epic glyph missing:\n%s", out)
	}
}

func TestTypeGlyphCellWidthIsUniform(t *testing.T) {
	t.Parallel()
	// A schema can declare any glyph, so the cell is normalised: a two-wide
	// emoji in one schema must not shift the column against a one-wide glyph
	// in another.
	for _, mode := range []datamodel.IconMode{datamodel.IconNerd, datamodel.IconEmoji, datamodel.IconText} {
		ic := iconSet{mode: mode}.withTypeIcons(map[string]string{"wide": "\U0001F680", "narrow": "@", "absent": ""})
		want := lipgloss.Width(ic.typeGlyph("narrow"))
		for _, typ := range []string{"wide", "absent"} {
			if got := lipgloss.Width(ic.typeGlyph(typ)); got != want {
				t.Errorf("mode %s: %s cell width = %d, want %d", mode, typ, got, want)
			}
		}
	}
}

func TestTreeNerdIconsSnapshot(t *testing.T) {
	t.Parallel()
	tm := loadedTree()
	golden.RequireEqual(t, []byte(tm.render(asciiTheme(), declaredIcons(datamodel.IconNerd), 100, 6, true, true)))
}

func TestTreeEmojiIconsSnapshot(t *testing.T) {
	t.Parallel()
	tm := loadedTree()
	golden.RequireEqual(t, []byte(tm.render(asciiTheme(), declaredIcons(datamodel.IconEmoji), 100, 6, true, true)))
}

func TestIconWidthsUniformPerMode(t *testing.T) {
	t.Parallel()
	groups := map[string][]glyph{
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

// The schema format carries one icon per type with no per-mode variants, so
// ui.icons selects state and priority glyphs but never the type glyph. A repo
// on a terminal without a nerd font sees the declared rune regardless; that is
// the accepted cost of letting a schema name any symbol it likes.
func TestShippedSchemaIconsIgnoreIconMode(t *testing.T) {
	schemas, err := entityschema.Load("")
	if err != nil {
		t.Fatal(err)
	}
	for typ, schema := range schemas {
		declared := schema.Representation.Icon
		if declared == "" {
			continue
		}
		for _, mode := range []datamodel.IconMode{datamodel.IconNerd, datamodel.IconEmoji, datamodel.IconText} {
			ic := iconSet{mode: mode}.withTypeIcons(map[string]string{typ: declared})
			if got := ic.typeGlyph(typ); got != declared {
				t.Errorf("type %q in mode %q: typeGlyph = %q, want the declared %q", typ, mode, got, declared)
			}
		}
	}
}

// The cell is uniform whether a glyph is padded or truncated, so uniformity
// alone cannot tell the two apart -- only the glyph surviving intact can.
func TestTypeGlyphKeepsTheDeclaredGlyphIntact(t *testing.T) {
	for _, icon := range []string{"@", "\uf41b", "[E]", "\U0001F680", "[EPIC]"} {
		ic := iconSet{mode: datamodel.IconNerd}.withTypeIcons(map[string]string{datamodel.TypeTicket: icon})

		if got := ic.typeGlyph(datamodel.TypeTicket); !strings.HasPrefix(got, icon) {
			t.Errorf("icon %q: typeGlyph = %q, want it to start with the declared glyph", icon, got)
		}
	}
}

func TestWithTypeIconsDoesNotAccumulateCellWidthAcrossCalls(t *testing.T) {
	ic := iconSet{mode: datamodel.IconNerd}.
		withTypeIcons(map[string]string{datamodel.TypeTicket: "\U0001F680"}).
		withTypeIcons(map[string]string{datamodel.TypeTicket: "@"})

	if got := ic.typeGlyph(datamodel.TypeTicket); got != "@" {
		t.Errorf("typeGlyph = %q, want %q -- the cell kept a width from a superseded icon set", got, "@")
	}
}
