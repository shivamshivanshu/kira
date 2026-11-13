package tui

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

func colorTheme() theme.Theme {
	r := lipgloss.NewRenderer(os.Stdout)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(true)
	return theme.New(r)
}

func bItem(id, num, title, state, cat string) datamodel.ListItem {
	return datamodel.ListItem{ID: id, Number: num, Title: title, Type: datamodel.TypeTicket, State: state, Category: cat}
}

func buildBoardResult() *datamodel.BoardResult {
	return &datamodel.BoardResult{Type: datamodel.TypeTicket, Columns: []datamodel.BoardColumn{
		{State: "TODO", Category: "todo", Count: 1, Items: []datamodel.ListItem{
			bItem("t1", "KIRA-150", "Add burst regression", "TODO", "todo"),
		}},
		{State: "IN_PROGRESS", Category: "doing", Wip: 3, Count: 4, Items: []datamodel.ListItem{
			bItem("t2", "KIRA-142", "Fix race in order-book snapshot merge", "IN_PROGRESS", "doing"),
			bItem("t3", "KIRA-160", "Dedup snapshots", "IN_PROGRESS", "doing"),
			bItem("t4", "KIRA-161", "Backpressure guard", "IN_PROGRESS", "doing"),
			bItem("t5", "KIRA-162", "Feed failover", "IN_PROGRESS", "doing"),
		}},
		{State: "REVIEW", Category: "doing", Wip: 2, Count: 1, Items: []datamodel.ListItem{
			bItem("t6", "KIRA-155", "Audit log format", "REVIEW", "doing"),
		}},
		{State: "DONE", Category: "done", Count: 1, Items: []datamodel.ListItem{
			bItem("t7", "KIRA-101", "Initial index", "DONE", "done"),
		}},
		{State: "WONT_DO", Category: "done", Items: []datamodel.ListItem{}},
	}}
}

func buildEmptyBoard() *datamodel.BoardResult {
	r := &datamodel.BoardResult{Type: datamodel.TypeTicket}
	for _, st := range []string{"TODO", "IN_PROGRESS", "REVIEW", "DONE", "WONT_DO"} {
		r.Columns = append(r.Columns, datamodel.BoardColumn{State: st, Items: []datamodel.ListItem{}})
	}
	return r
}

func newBoardTestModel(w, h int, cfg *datamodel.Config, res *datamodel.BoardResult) (model, *boardScreen) {
	m := newModel(nil, cfg, asciiTheme(), iconSet{mode: datamodel.IconText}, false)
	m.width, m.height = w, h
	m.view = viewBoard
	bs := m.screens[viewBoard].(*boardScreen)
	bs.state = loadReady
	bs.board.load(res)
	return m, bs
}

func TestBoardView(t *testing.T) {
	t.Parallel()
	m, _ := newBoardTestModel(100, 12, config.Default(), buildBoardResult())
	golden.RequireEqual(t, []byte(m.View()))
}

func TestBoardEmptyState(t *testing.T) {
	t.Parallel()
	m, _ := newBoardTestModel(100, 12, config.Default(), buildEmptyBoard())
	golden.RequireEqual(t, []byte(m.View()))
}

func TestBoardTeatestSnapshot(t *testing.T) {
	t.Parallel()
	m, _ := newBoardTestModel(100, 12, config.Default(), buildBoardResult())
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 12))
	tm.Type("l")
	tm.Type("j")
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t)
	golden.RequireEqual(t, []byte(tm.FinalModel(t).(model).View()))
}

func buildMixedBoardResult() *datamodel.BoardResult {
	card := func(id, num, board string) datamodel.ListItem {
		it := bItem(id, num, num, "TODO", "todo")
		it.Board = board
		return it
	}
	return &datamodel.BoardResult{Type: datamodel.TypeTicket, Columns: []datamodel.BoardColumn{
		{State: "TODO", Category: "todo", Count: 3, Items: []datamodel.ListItem{
			card("a", "KIRA-1", "KIRA"),
			card("b", "XYZ-1", "XYZ"),
			card("c", "KIRA-2", "KIRA"),
		}},
	}}
}

func TestBoardScopeKeepsGlobalCount(t *testing.T) {
	scoped := scopedBoard(buildMixedBoardResult(), "xyz")
	items := scoped.Columns[0].Items
	if len(items) != 1 || items[0].Number != "XYZ-1" {
		t.Fatalf("scope xyz should keep only XYZ-1, got %+v", items)
	}
	if scoped.Columns[0].Count != 3 {
		t.Fatalf("scoped column Count should stay global (3), got %d", scoped.Columns[0].Count)
	}
}

func TestBoardScopeHeaderHidesOtherBoards(t *testing.T) {
	m, bs := newBoardTestModel(100, 12, config.Default(), buildMixedBoardResult())
	bs.raw = buildMixedBoardResult()
	bs.scope = "XYZ"
	bs.applyScope()
	view := m.View()
	if !strings.Contains(view, "board: XYZ") {
		t.Fatalf("scoped board view must show the scope header:\n%s", view)
	}
	if !strings.Contains(view, "XYZ-1") {
		t.Fatalf("scoped view should show the in-scope card:\n%s", view)
	}
	if strings.Contains(view, "KIRA-1") || strings.Contains(view, "KIRA-2") {
		t.Fatalf("scoped view must hide other boards' cards:\n%s", view)
	}
}

func TestBoardScopePickerSelectsBoard(t *testing.T) {
	cfg := config.Default()
	cfg.Boards = []datamodel.Board{
		{Key: "KIRA", Name: "Kira", Default: true},
		{Key: "XYZ", Name: "Exchange"},
	}
	m, bs := newBoardTestModel(100, 12, cfg, buildMixedBoardResult())
	bs.raw = buildMixedBoardResult()

	up, _ := m.Update(key("b"))
	m = up.(model)
	if m.picker == nil || len(m.picker.entries) != 3 {
		t.Fatalf("b should open a picker with All + 2 boards, got %+v", m.picker)
	}
	up, _ = m.Update(key("j"))
	m = up.(model)
	up, _ = m.Update(key("j"))
	m = up.(model)
	up, _ = m.Update(key("enter"))
	m = up.(model)
	if m.picker != nil {
		t.Fatal("enter should close the picker")
	}
	if bs.scope != "XYZ" {
		t.Fatalf("selecting the XYZ entry should set scope=XYZ, got %q", bs.scope)
	}
	if _, ok := bs.board.selected(); !ok {
		t.Fatal("scoped board should still have the XYZ card selectable")
	}
}

func TestBoardMoveUnderScopeKeepsScopeAndSnapshot(t *testing.T) {
	t.Parallel()
	m, bs := newBoardTestModel(100, 12, config.Default(), buildMixedBoardResult())
	bs.raw = buildMixedBoardResult()
	bs.scope = "XYZ"
	bs.applyScope()

	card := func(id, num, board, state string) datamodel.ListItem {
		it := bItem(id, num, num, state, "todo")
		it.Board = board
		return it
	}
	moved := &datamodel.BoardResult{Type: datamodel.TypeTicket, Columns: []datamodel.BoardColumn{
		{State: "TODO", Category: "todo", Count: 2, Items: []datamodel.ListItem{
			card("a", "KIRA-1", "KIRA", "TODO"),
			card("c", "KIRA-2", "KIRA", "TODO"),
		}},
		{State: "IN_PROGRESS", Category: "doing", Count: 1, Items: []datamodel.ListItem{
			card("b", "XYZ-1", "XYZ", "IN_PROGRESS"),
		}},
	}}
	bs.applyMove(&m, boardMovedMsg{
		res:    &datamodel.MoveResult{Number: "XYZ-1", From: "TODO", To: "IN_PROGRESS"},
		board:  moved,
		cardID: "b",
	})

	if bs.raw != moved {
		t.Fatal("applyMove must adopt the reloaded board as the raw snapshot")
	}
	view := m.View()
	if strings.Contains(view, "KIRA-1") || strings.Contains(view, "KIRA-2") {
		t.Fatalf("post-move board must stay scoped to XYZ:\n%s", view)
	}
	if strings.Count(view, "XYZ-1") < 2 {
		t.Fatalf("moved in-scope card must render in its column, not just the move notice:\n%s", view)
	}
	if sel, ok := bs.board.selected(); !ok || sel.ID != "b" {
		t.Fatalf("moved card should stay focused, got %+v ok=%v", sel, ok)
	}
}

func TestColumnHeaderTintReflectsWipPressure(t *testing.T) {
	t.Parallel()
	th := colorTheme()
	over := datamodel.BoardColumn{State: "IN_PROGRESS", Wip: 3, Count: 4}
	atLimit := datamodel.BoardColumn{State: "IN_PROGRESS", Wip: 3, Count: 3}
	within := datamodel.BoardColumn{State: "TODO"}
	const probe = "x"
	if columnHeaderStyle(th, over, false).Render(probe) != th.Heat.Hot.Render(probe) {
		t.Error("column over its WIP limit must tint with Heat.Hot")
	}
	if columnHeaderStyle(th, atLimit, false).Render(probe) != th.Heat.Warm.Render(probe) {
		t.Error("column at its WIP limit must tint with Heat.Warm")
	}
	if columnHeaderStyle(th, within, false).Render(probe) != th.Dim.Render(probe) {
		t.Error("unfocused within-limit column must render Dim")
	}
	if columnHeaderStyle(th, within, true).Render(probe) != th.Accent.Bold(true).Render(probe) {
		t.Error("focused column must render Accent bold")
	}
	if columnLabel(over) != "IN_PROGRESS  4/3" {
		t.Errorf("n/wip header = %q, want IN_PROGRESS  4/3", columnLabel(over))
	}
	if columnLabel(within) != "TODO" {
		t.Errorf("no-wip header = %q, want TODO", columnLabel(within))
	}
}

func TestCardWindow(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                                       string
		total, capacity, focusRow                  int
		wantStart, wantSlots, wantAbove, wantBelow int
	}{
		{"all fit", 3, 5, 0, 0, 3, 0, 0},
		{"exact fit", 5, 5, 4, 0, 5, 0, 0},
		{"unfocused column tops out", 10, 5, -1, 0, 4, 0, 6},
		{"top window", 10, 5, 0, 0, 4, 0, 6},
		{"top window last slot", 10, 5, 3, 0, 4, 0, 6},
		{"middle both indicators", 10, 5, 4, 2, 3, 2, 5},
		{"middle shifted", 10, 5, 5, 3, 3, 3, 4},
		{"bottom window start", 10, 5, 6, 6, 4, 6, 0},
		{"bottom focus fills all slots", 10, 5, 9, 6, 4, 6, 0},
		{"tiny middle", 5, 3, 2, 2, 1, 2, 2},
		{"capacity one follows focus", 10, 1, 5, 5, 1, 0, 0},
		{"capacity two at end", 10, 2, 9, 8, 2, 0, 0},
		{"capacity two at start", 10, 2, 0, 0, 2, 0, 0},
		{"no items", 0, 5, 0, 0, 0, 0, 0},
		{"no capacity", 10, 0, 0, 0, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, slots, above, below := cardWindow(tc.total, tc.capacity, tc.focusRow)
			if start != tc.wantStart || slots != tc.wantSlots || above != tc.wantAbove || below != tc.wantBelow {
				t.Fatalf("cardWindow(%d,%d,%d) = (%d,%d,%d,%d), want (%d,%d,%d,%d)",
					tc.total, tc.capacity, tc.focusRow, start, slots, above, below,
					tc.wantStart, tc.wantSlots, tc.wantAbove, tc.wantBelow)
			}
			lines := slots
			if above > 0 {
				lines++
			}
			if below > 0 {
				lines++
			}
			if tc.capacity > 0 && lines > tc.capacity {
				t.Fatalf("window uses %d lines, exceeding capacity %d", lines, tc.capacity)
			}
			if tc.focusRow >= 0 && tc.focusRow < tc.total && slots > 0 &&
				(tc.focusRow < start || tc.focusRow >= start+slots) {
				t.Fatalf("focused row %d outside window [%d,%d)", tc.focusRow, start, start+slots)
			}
		})
	}
}

func TestBoardBottomFocusShowsAboveIndicatorNotBlank(t *testing.T) {
	t.Parallel()
	items := make([]datamodel.ListItem, 10)
	for i := range items {
		items[i] = bItem("id"+strconv.Itoa(i), "KIRA-"+strconv.Itoa(i), "T"+strconv.Itoa(i), "TODO", "todo")
	}
	res := &datamodel.BoardResult{Type: datamodel.TypeTicket, Columns: []datamodel.BoardColumn{
		{State: "TODO", Category: "todo", Count: 10, Items: items},
	}}
	out := renderBoard(asciiTheme(), iconSet{mode: datamodel.IconText}, res, 40, 7, 0, 9)
	if !strings.Contains(out, "+6 above") {
		t.Errorf("bottom-focused overflowing column must show the above-window indicator:\n%s", out)
	}
	if !strings.Contains(out, "KIRA-9") {
		t.Errorf("focused last card must be rendered:\n%s", out)
	}
	if strings.Contains(out, "more") {
		t.Errorf("nothing is hidden below; no below indicator expected:\n%s", out)
	}
}

func TestBoardQQuitsInsteadOfJumpingToTree(t *testing.T) {
	t.Parallel()
	m, bs := newBoardTestModel(100, 12, config.Default(), buildBoardResult())

	bs.peek = peekDocked
	u, cmd := m.Update(key("q"))
	m = u.(model)
	if bs.peek != peekOff {
		t.Fatal("q must close the peek pane first")
	}
	if cmd != nil {
		t.Fatal("closing the peek pane must not quit")
	}

	u, cmd = m.Update(key("q"))
	m = u.(model)
	if m.view != viewBoard {
		t.Fatalf("q on the board must not bounce to the tree, view=%v", m.view)
	}
	if cmd == nil {
		t.Fatal("q on the board with no peek must quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("q must return a quit command, got %T", cmd())
	}
}

func TestBoardMoveErrorNoticeIsFirstLineAndHot(t *testing.T) {
	t.Parallel()
	m, bs := newBoardTestModel(100, 12, config.Default(), buildBoardResult())
	bs.applyMove(&m, boardMovedMsg{err: errors.New("boom first line\n  context second line")})
	if bs.notice != "boom first line" {
		t.Fatalf("error notice = %q, want the first non-empty line only", bs.notice)
	}
	if !bs.noticeErr {
		t.Fatal("a failed move must mark the notice as an error")
	}
	bs.applyMove(&m, boardMovedMsg{
		res:    &datamodel.MoveResult{Number: "KIRA-1", From: "TODO", To: "IN_PROGRESS"},
		board:  buildBoardResult(),
		cardID: "t1",
	})
	if bs.noticeErr {
		t.Fatal("a successful move must clear the error mark")
	}
}

func boardStore(t *testing.T) (*core.Store, *datamodel.Config) {
	t.Helper()
	s, cfg, _ := initRepo(t)
	return s, cfg
}

func drainBoardLoad(t *testing.T, m *model, bs *boardScreen) {
	t.Helper()
	cmd := bs.activate(m)
	if cmd == nil {
		t.Fatal("activate must dispatch a board load")
	}
	msg, ok := cmd().(boardLoadedMsg)
	if !ok {
		t.Fatalf("board load must resolve to boardLoadedMsg, got %T", cmd())
	}
	bs.applyLoaded(msg)
	m.drain()
}

func TestBoardViewIsPureAndLoadsAsync(t *testing.T) {
	t.Parallel()
	s, cfg := boardStore(t)
	cr, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "A", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(s, cfg, asciiTheme(), iconSet{mode: datamodel.IconText}, false)
	m.width, m.height = 100, 12

	u, cmd := m.Update(key("2"))
	m = u.(model)
	bs := m.screens[viewBoard].(*boardScreen)
	if cmd == nil || bs.state != loadPending {
		t.Fatal("switching to the board must dispatch an async load and mark pending")
	}
	view := bs.view(&m, 100, 10)
	if strings.Contains(view, cr.Number) {
		t.Fatalf("view must not read the store; card rendered before the load resolved:\n%s", view)
	}
	if !strings.Contains(view, "loading") {
		t.Fatalf("pending board must render the loading placeholder, got:\n%s", view)
	}
	if again := bs.activate(&m); again != nil {
		t.Fatal("activate must not dispatch a second load while one is pending")
	}

	msg, ok := cmd().(boardLoadedMsg)
	if !ok {
		t.Fatalf("board load must resolve to boardLoadedMsg, got %T", cmd())
	}
	u, _ = m.Update(msg)
	m = u.(model)
	if view := bs.view(&m, 100, 10); !strings.Contains(view, cr.Number) {
		t.Fatalf("resolved load must render the card:\n%s", view)
	}
}

func TestBoardMoveHitsCoreMovePath(t *testing.T) {
	t.Parallel()
	s, cfg := boardStore(t)
	cr, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "M", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(s, cfg, asciiTheme(), iconSet{mode: datamodel.IconText}, false)
	m.width, m.height = 100, 12
	m.view = viewBoard
	bs := m.screens[viewBoard].(*boardScreen)
	drainBoardLoad(t, &m, bs)
	bs.board.focusByID(cr.ID)

	u, cmd := m.Update(key("L"))
	m = u.(model)
	if cmd == nil {
		t.Fatal("L on a TODO card must dispatch a move command")
	}
	m.Update(cmd())
	if bs.notice == "" {
		t.Fatal("board move produced no notice")
	}
	if state := stateOnDisk(t, s, cfg, cr.ID); state != "IN_PROGRESS" {
		t.Fatalf("board move did not reach core.Move: on-disk state = %s", state)
	}
	out := gitOutput(t, s.Root(), "log", "--format=%s", "-1")
	if !strings.Contains(out, "kira: "+cr.Number+" state TODO -> IN_PROGRESS") {
		t.Fatalf("commit subject %q is not the core.Move subject only that path writes", strings.TrimSpace(out))
	}
}

func TestBoardMoveOutOfDoneLandsInNextColumn(t *testing.T) {
	t.Parallel()
	s, cfg := boardStore(t)
	cr, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "D", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Move(cfg, cr.ID, "DONE", core.MoveOpts{Force: true}); err != nil {
		t.Fatal(err)
	}
	m := newModel(s, cfg, asciiTheme(), iconSet{mode: datamodel.IconText}, false)
	m.width, m.height = 100, 12
	m.view = viewBoard
	bs := m.screens[viewBoard].(*boardScreen)
	drainBoardLoad(t, &m, bs)
	bs.board.focusByID(cr.ID)

	_, cmd := m.Update(key("L"))
	if cmd == nil {
		t.Fatal("moving out of DONE must dispatch a move, not be greyed out")
	}
	cmd()
	if state := stateOnDisk(t, s, cfg, cr.ID); state != "WONT_DO" {
		t.Fatalf("state = %s, want WONT_DO: every declared state is reachable now", state)
	}
}

func TestBoardPeekMountsDetailComponent(t *testing.T) {
	t.Parallel()
	m, bs := newBoardTestModel(100, 12, config.Default(), buildBoardResult())
	bs.update(&m, "p")
	if bs.peek != peekDocked {
		t.Fatalf("p should dock the peek pane, mode = %d", bs.peek)
	}
	if view := bs.view(&m, 100, 10); !strings.Contains(view, "Select an item") {
		t.Fatalf("docked peek should mount the detail component, got:\n%s", view)
	}
	bs.update(&m, "enter")
	if bs.peek != peekOverlay {
		t.Fatalf("enter should open the overlay, mode = %d", bs.peek)
	}
	if k := bs.keys(); len(k) == 0 || k[0].Key != "j/k" || k[0].Desc != "scroll" {
		t.Fatalf("overlay peek should surface the detail component's keymap, got %+v", k)
	}
	before := bs.board.row
	bs.update(&m, "j")
	if bs.board.row != before {
		t.Fatal("j in the overlay must scroll the panel, not move the board cursor")
	}
}

func TestJumpFromBoardRecordsSelectedCard(t *testing.T) {
	t.Parallel()
	m, bs := newBoardTestModel(100, 12, config.Default(), buildBoardResult())
	bs.board.focusByID("t3")

	updated, _ := m.Update(key("1"))
	m = updated.(model)
	if m.view != viewTree {
		t.Fatalf("1 should switch to the tree view, got %v", m.view)
	}
	if len(m.jumps.entries) == 0 {
		t.Fatal("switching away from the board should record a jump entry")
	}
	last := m.jumps.entries[len(m.jumps.entries)-1]
	if last.view != viewBoard || last.itemID != "t3" {
		t.Fatalf("jump from board should record the selected card, got %+v", last)
	}
}

func stateOnDisk(t *testing.T, s *core.Store, cfg *datamodel.Config, id string) string {
	t.Helper()
	show, err := s.Show(cfg, id, "")
	if err != nil {
		t.Fatal(err)
	}
	return show.State
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func TestBoardCardPlumbsPriorityAndResolution(t *testing.T) {
	t.Parallel()
	p0 := "P0"
	dropped := datamodel.ResolutionDropped
	res := &datamodel.BoardResult{Type: datamodel.TypeTicket, Columns: []datamodel.BoardColumn{
		{State: "WONT_DO", Category: "done", Count: 1, Items: []datamodel.ListItem{
			{ID: "d1", Number: "KIRA-9", Title: "Dropped work", Type: datamodel.TypeTicket, State: "WONT_DO", Category: "done", Priority: &p0, Resolution: &dropped},
		}},
	}}
	out := renderBoard(asciiTheme(), iconSet{mode: datamodel.IconText, priorities: []string{"P0", "P1", "P2", "P3"}}, res, 40, 4, -1, -1)
	if !strings.Contains(out, "!") {
		t.Errorf("P0 priority marker missing from card:\n%s", out)
	}
	if !strings.Contains(out, "[-]") {
		t.Errorf("dropped glyph missing from card:\n%s", out)
	}
	if strings.Contains(out, "[x]") {
		t.Errorf("dropped card must not render the done glyph:\n%s", out)
	}
}

func TestBoardCardTruncatesInsteadOfWrapping(t *testing.T) {
	t.Parallel()
	card := bItem("a", "CORE-40", "Long enough title", "IN_PROGRESS", "doing")
	card.Priority = new("P0")
	res := &datamodel.BoardResult{Type: datamodel.TypeTicket, Columns: []datamodel.BoardColumn{
		{State: "TODO", Category: "todo"},
		{State: "IN_PROGRESS", Category: "doing", Count: 1, Items: []datamodel.ListItem{card}},
		{State: "REVIEW", Category: "doing"},
		{State: "DONE", Category: "done"},
		{State: "WONT_DO", Category: "done"},
	}}
	ic := iconSet{mode: datamodel.IconText, priorities: []string{"P0", "P1", "P2"}}
	out := renderBoard(asciiTheme(), ic, res, 80, 3, 1, 0)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("a card must render as exactly one line; board of height 3 came out %d lines:\n%s", len(lines), out)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w > 80 {
			t.Fatalf("line %d is %d cells wide, budget 80:\n%s", i, w, out)
		}
	}
	if !strings.Contains(out, "CORE-") {
		t.Fatalf("truncated card must keep the leading segments visible:\n%s", out)
	}
}

func TestBoardMoveRefreshesTree(t *testing.T) {
	t.Parallel()
	s, cfg := boardStore(t)
	cr, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "M", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(s, cfg, asciiTheme(), iconSet{mode: datamodel.IconText}, false)
	m.width, m.height = 100, 12
	m.view = viewBoard
	bs := m.screens[viewBoard].(*boardScreen)
	drainBoardLoad(t, &m, bs)
	bs.board.focusByID(cr.ID)

	u, mv := m.Update(key("L"))
	m = u.(model)
	if mv == nil {
		t.Fatal("L on a TODO card must dispatch a move command")
	}
	u, cmd := m.Update(mv())
	m = u.(model)
	if cmd == nil {
		t.Fatal("a successful board move must enqueue a tree refresh; without one the tree stays stale forever under the default no-auto-refresh config")
	}
	msg, ok := cmd().(treeLoadedMsg)
	if !ok {
		t.Fatalf("the enqueued refresh must resolve to treeLoadedMsg, got %T", cmd())
	}
	u, _ = m.Update(msg)
	m = u.(model)
	ts := m.screens[viewTree].(*treeScreen)
	if got := ts.tree.fields[cr.ID].State; got != "IN_PROGRESS" {
		t.Fatalf("tree must show the moved state after the refresh, got %q", got)
	}

	if _, cmd := m.Update(boardMovedMsg{err: errStub("boom"), cardID: cr.ID}); cmd != nil {
		t.Fatal("a failed move must not enqueue a refresh")
	}
}
