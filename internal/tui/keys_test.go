package tui

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestTreeGgGJumpsTopAndBottom(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	ts := m.screens[viewTree].(*treeScreen)
	last := len(ts.tree.rows) - 1
	ts.tree.cursor = last

	ts.update(&m, "g")
	ts.update(&m, "g")
	if ts.tree.cursor != 0 {
		t.Fatalf("gg should land on the first row, cursor=%d", ts.tree.cursor)
	}

	ts.update(&m, "G")
	if ts.tree.cursor != last {
		t.Fatalf("G should land on the last row, cursor=%d want %d", ts.tree.cursor, last)
	}
}

func TestTreeGpSurvivesChordExtension(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	ts := m.screens[viewTree].(*treeScreen)
	ts.tree.cursor = 2 // KIRA-142, a child of E1

	ts.update(&m, "g")
	ts.update(&m, "p")
	if got := ts.tree.selectedID(); got != "E1" {
		t.Fatalf("gp should still jump to the parent epic, selected=%q", got)
	}
}

func TestTreeUnknownGChordIsNoOp(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	ts := m.screens[viewTree].(*treeScreen)
	ts.tree.cursor = 1

	ts.update(&m, "g")
	ts.update(&m, "x")
	if ts.tree.cursor != 1 {
		t.Fatalf("an unrecognized second key must not move the cursor, cursor=%d", ts.tree.cursor)
	}
	if ts.pendingG {
		t.Fatal("the g chord must be dropped after the second key")
	}
}

func TestTreeCollapseAllSnapsCursorAndExpandAllRestores(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	ts := m.screens[viewTree].(*treeScreen)
	full := len(ts.tree.rows)
	ts.tree.cursor = 1 // inside the fold, a child of E1

	ts.update(&m, "z")
	ts.update(&m, "M")
	if len(ts.tree.rows) != 1 {
		t.Fatalf("zM should collapse every epic to its top-level row, rows=%d", len(ts.tree.rows))
	}
	if got := ts.tree.selectedID(); got != "E1" {
		t.Fatalf("zM should snap the cursor to the top-level ancestor, selected=%q", got)
	}

	ts.update(&m, "z")
	ts.update(&m, "R")
	if len(ts.tree.rows) != full {
		t.Fatalf("zR should expand back to %d rows, rows=%d", full, len(ts.tree.rows))
	}
	if got := ts.tree.selectedID(); got != "E1" {
		t.Fatalf("zR should keep the cursor on the same node, selected=%q", got)
	}
}

func TestTreeDetailPaneForwardsDetailKeysAndFallsThrough(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 40, true)
	ts := m.screens[viewTree].(*treeScreen)
	ts.host.cache["E1"] = sampleDetail()
	ts.tree.cursor = 0
	ts.focus = paneDetail
	ts.syncDetail(&m)

	cursorBefore := ts.tree.cursor
	ts.update(&m, "j")
	if ts.host.panel.scroll == 0 {
		t.Fatal("j is a detail key; it should scroll the detail panel")
	}
	if ts.tree.cursor != cursorBefore {
		t.Fatalf("a forwarded detail key must not also move the tree cursor, cursor=%d", ts.tree.cursor)
	}

	ts.update(&m, "tab")
	if ts.focus != paneTree {
		t.Fatal("tab is not a detail key; it must fall through to toggle focus back to the tree")
	}
}

func TestBoardGgGJumpsTopAndBottom(t *testing.T) {
	t.Parallel()
	m, bs := newBoardTestModel(100, 12, config.Default(), buildBoardResult())
	bs.board.col = 1 // IN_PROGRESS has four cards
	bs.board.row = 2
	last := bs.board.colLen() - 1

	bs.update(&m, "g")
	bs.update(&m, "g")
	if bs.board.row != 0 {
		t.Fatalf("gg should land on the first card, row=%d", bs.board.row)
	}

	bs.update(&m, "G")
	if bs.board.row != last {
		t.Fatalf("G should land on the last card, row=%d want %d", bs.board.row, last)
	}
}

func TestDetailHalfPageMovesAndClampsBothEnds(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 20, true)
	d := newDetailPanel()
	res := sampleDetail()
	half := m.mainHeight() / 2

	d.update(&m, res, "ctrl+d")
	if d.scroll != half {
		t.Fatalf("ctrl+d should scroll half a page, scroll=%d want %d", d.scroll, half)
	}

	for i := 0; i < 50; i++ {
		d.update(&m, res, "ctrl+d")
	}
	d.render(asciiTheme(), iconSet{mode: datamodel.IconText}, res, 100, 8)
	lines := d.contentLines(asciiTheme(), iconSet{mode: datamodel.IconText}, res, 100)
	if want := max(0, len(lines)-8); d.scroll != want {
		t.Fatalf("ctrl+d should clamp at the bottom, scroll=%d want %d", d.scroll, want)
	}

	for i := 0; i < 50; i++ {
		d.update(&m, res, "ctrl+u")
	}
	d.render(asciiTheme(), iconSet{mode: datamodel.IconText}, res, 100, 8)
	if d.scroll != 0 {
		t.Fatalf("ctrl+u should clamp at the top, scroll=%d want 0", d.scroll)
	}
}

func TestStatsGgGAndHalfPageScroll(t *testing.T) {
	t.Parallel()
	m, ss := statsScreenWith(sampleStats())
	ss.scroll = 3

	ss.update(&m, "g")
	ss.update(&m, "g")
	if ss.scroll != 0 {
		t.Fatalf("gg should scroll stats to the top, scroll=%d", ss.scroll)
	}

	ss.update(&m, "ctrl+d")
	if want := m.mainHeight() / 2; ss.scroll != want {
		t.Fatalf("ctrl+d should scroll half a page, scroll=%d want %d", ss.scroll, want)
	}
}
