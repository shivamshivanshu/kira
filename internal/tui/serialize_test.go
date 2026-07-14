package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestMutationThenRefreshSerializes(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)

	if cmd := m.request(func() tea.Msg { return treeLoadedMsg{} }); cmd == nil {
		t.Fatal("first store command must dispatch immediately")
	}
	if !m.busy {
		t.Fatal("model must be busy while a mutation is in flight")
	}

	u, cmd := m.Update(key("r"))
	m = u.(model)
	if cmd != nil {
		t.Fatal("refresh must not run while a mutation is in flight")
	}
	if len(m.pending) != 1 {
		t.Fatalf("refresh must be queued, pending=%d", len(m.pending))
	}

	u, cmd = m.Update(treeLoadedMsg{})
	m = u.(model)
	if cmd == nil {
		t.Fatal("mutation completion must release the queued refresh")
	}
	if len(m.pending) != 0 || !m.busy {
		t.Fatalf("released refresh must be the sole in-flight op: pending=%d busy=%v", len(m.pending), m.busy)
	}

	u, cmd = m.Update(treeLoadedMsg{})
	m = u.(model)
	if cmd != nil || m.busy {
		t.Fatalf("executor must go idle after draining the queue: cmd=%v busy=%v", cmd != nil, m.busy)
	}
}

func TestQuitDefersWhileBusy(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	if cmd := m.request(func() tea.Msg { return treeLoadedMsg{} }); cmd == nil {
		t.Fatal("mutation should be in flight")
	}

	u, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = u.(model)
	if cmd != nil {
		t.Fatal("ctrl+c while busy must defer the quit, not sever the mutation")
	}
	if !m.quitting {
		t.Fatal("first ctrl+c while busy must arm quitting")
	}

	u, cmd = m.Update(treeLoadedMsg{})
	m = u.(model)
	if cmd == nil {
		t.Fatal("quit must fire once the in-flight mutation settles")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("deferred quit must return QuitMsg at idle, got %T", cmd())
	}
}

func TestSecondCtrlCForceQuits(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	m.request(func() tea.Msg { return treeLoadedMsg{} })

	u, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = u.(model)
	u, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = u.(model)
	if cmd == nil {
		t.Fatal("a second ctrl+c must force-quit immediately")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("forced quit must return QuitMsg, got %T", cmd())
	}
}

func TestBoardPeekDefersAndReplaysDetailSync(t *testing.T) {
	t.Parallel()
	s, cfg := boardStore(t)
	a, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "A", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "B", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}

	m := newModel(s, cfg, asciiTheme(), iconSet{mode: datamodel.IconText}, false)
	m.width, m.height = 100, 12
	m.view = viewBoard
	bs := m.screens[viewBoard].(*boardScreen)
	bs.ensureLoaded(&m)
	bs.peek = peekDocked
	bs.board.focusByID(a.ID)
	bs.syncDetail(&m)
	if bs.host.detail == nil || bs.host.detail.ID != a.ID {
		t.Fatalf("peek should show A, got %+v", bs.host.detail)
	}

	m.busy = true
	bs.board.focusByID(b.ID)
	bs.syncDetail(&m)
	if !bs.host.dirty {
		t.Fatal("a detail sync during an in-flight mutation must be deferred")
	}
	if bs.host.detail.ID != a.ID {
		t.Fatal("detail must stay on A while the mutation is in flight")
	}

	m.busy = false
	bs.settle(&m)
	if bs.host.dirty {
		t.Fatal("settle must clear the dirty flag")
	}
	if bs.host.detail == nil || bs.host.detail.ID != b.ID {
		t.Fatalf("settle must replay the deferred sync to B, got %+v", bs.host.detail)
	}
}

func TestBoardReloadsAfterTreeLoaded(t *testing.T) {
	t.Parallel()
	s, cfg := boardStore(t)
	if _, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "A", NoEdit: true}); err != nil {
		t.Fatal(err)
	}

	m := newModel(s, cfg, asciiTheme(), iconSet{mode: datamodel.IconText}, false)
	m.width, m.height = 100, 12
	m.view = viewBoard
	bs := m.screens[viewBoard].(*boardScreen)
	bs.ensureLoaded(&m)
	if !bs.loaded {
		t.Fatal("board must latch loaded after the first ensureLoaded")
	}
	if got := boardCardCount(bs); got != 1 {
		t.Fatalf("board must show the sole ticket, got %d", got)
	}

	if _, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "B", NoEdit: true}); err != nil {
		t.Fatal(err)
	}

	u, _ := m.Update(treeLoadedMsg{})
	m = u.(model)
	if bs.loaded {
		t.Fatal("a treeLoadedMsg must invalidate the board latch")
	}

	bs.ensureLoaded(&m)
	if got := boardCardCount(bs); got != 2 {
		t.Fatalf("board must re-read after invalidation and show both tickets, got %d", got)
	}
}

func boardCardCount(bs *boardScreen) int {
	n := 0
	for _, c := range bs.board.columns() {
		n += len(c.Items)
	}
	return n
}

func TestTreeDetailReplaysAtDrainIdle(t *testing.T) {
	t.Parallel()
	s, cfg := boardStore(t)
	cr, err := s.Create(cfg, core.CreateOpts{Type: datamodel.TypeTicket, Title: "T", NoEdit: true})
	if err != nil {
		t.Fatal(err)
	}

	m := newModel(s, cfg, asciiTheme(), iconSet{mode: datamodel.IconText}, false)
	m.width, m.height = 100, 12
	ts := m.screens[viewTree].(*treeScreen)

	if cmd := m.request(func() tea.Msg { return commandResultMsg{} }); cmd == nil {
		t.Fatal("op should be in flight")
	}
	ts.host.sync(&m, cr.ID)
	if !ts.host.dirty {
		t.Fatal("a detail sync during busy must be deferred")
	}

	u, _ := m.Update(commandResultMsg{})
	m = u.(model)
	if ts.host.dirty {
		t.Fatal("draining to idle must replay the deferred detail sync")
	}
	if ts.host.detail == nil || ts.host.detail.ID != cr.ID {
		t.Fatalf("settle must load the deferred detail, got %+v", ts.host.detail)
	}
}
