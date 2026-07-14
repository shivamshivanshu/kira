package tui

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestPruneNodesKeepsMatchesAndAncestors(t *testing.T) {
	t.Parallel()
	nodes := []datamodel.TreeNode{
		{ID: "E1", Children: []datamodel.TreeNode{{ID: "T1"}, {ID: "T2"}}},
		{ID: "E2", Children: []datamodel.TreeNode{{ID: "T3"}}},
	}
	got := pruneNodes(nodes, map[string]bool{"T1": true})
	if len(got) != 1 || got[0].ID != "E1" {
		t.Fatalf("want only E1 (ancestor of match), got %+v", got)
	}
	if len(got[0].Children) != 1 || got[0].Children[0].ID != "T1" {
		t.Fatalf("want only matched child T1, got %+v", got[0].Children)
	}
}

func TestLoadFilteredTreeNarrowsToMatch(t *testing.T) {
	t.Parallel()
	s, cfg, _ := initRepo(t)
	alpha := createTicket(t, s, cfg, "alpha ticket")
	createTicket(t, s, cfg, "beta ticket")

	data, err := loadFilteredTree(s, cfg, "alpha")
	if err != nil {
		t.Fatalf("loadFilteredTree: %v", err)
	}
	if len(data.nodes) != 1 || data.nodes[0].Number != alpha {
		t.Fatalf("filter should narrow to %s, got %+v", alpha, data.nodes)
	}
}

func TestApplyFilterEmitsTreeLoadedMsg(t *testing.T) {
	t.Parallel()
	s, cfg, _ := initRepo(t)
	alpha := createTicket(t, s, cfg, "alpha ticket")
	createTicket(t, s, cfg, "beta ticket")

	m := newModel(s, cfg, asciiTheme(), iconSet{}, false)
	m.width, m.height = 100, 12
	m.barRoute(key("/"))
	typeInto(&m, "alpha")
	cmd, done := m.barRoute(enter())
	if !done || cmd == nil {
		t.Fatalf("filter submit should be handled and emit a load cmd (done=%v cmd=%v)", done, cmd)
	}
	msg, ok := cmd().(treeLoadedMsg)
	if !ok {
		t.Fatalf("filter cmd should yield treeLoadedMsg, got %T", cmd())
	}
	if msg.err != nil {
		t.Fatalf("filter load error: %v", msg.err)
	}
	if len(msg.data.nodes) != 1 || msg.data.nodes[0].Number != alpha {
		t.Fatalf("filtered tree should contain only %s, got %+v", alpha, msg.data.nodes)
	}
	if m.bar.filter != "alpha" {
		t.Fatalf("active filter = %q, want alpha", m.bar.filter)
	}
}

func TestCommandRefreshKeepsActiveFilter(t *testing.T) {
	t.Parallel()
	s, cfg, _ := initRepo(t)
	alpha := createTicket(t, s, cfg, "alpha ticket")
	createTicket(t, s, cfg, "beta ticket")

	m := newModel(s, cfg, asciiTheme(), iconSet{}, false)
	m.width, m.height = 100, 12
	m.barRoute(key("/"))
	typeInto(&m, "alpha")
	m.barRoute(enter())

	_, cmd := m.Update(commandResultMsg{text: "Moved", refresh: true})
	if cmd == nil {
		t.Fatal("a refreshing command result should issue a reload")
	}
	msg, ok := cmd().(treeLoadedMsg)
	if !ok {
		t.Fatalf("reload should yield treeLoadedMsg, got %T", cmd())
	}
	if len(msg.data.nodes) != 1 || msg.data.nodes[0].Number != alpha {
		t.Fatalf("command refresh dropped the active filter: %+v", msg.data.nodes)
	}
	if m.bar.filter != "alpha" {
		t.Fatalf("footer filter state lost after command: %q", m.bar.filter)
	}
}
