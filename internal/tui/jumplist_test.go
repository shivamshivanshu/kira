package tui

import (
	"strconv"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestJumplistPushDedupsConsecutiveDuplicateOnly(t *testing.T) {
	t.Parallel()
	var j jumplist
	j.push(jumpEntry{view: viewTree, itemID: "T1"})
	j.push(jumpEntry{view: viewTree, itemID: "T1"})
	if len(j.entries) != 1 {
		t.Fatalf("len = %d, want 1 (consecutive duplicate must dedup)", len(j.entries))
	}
	j.push(jumpEntry{view: viewBoard, itemID: "T1"})
	if len(j.entries) != 2 {
		t.Fatalf("len = %d, want 2 (differing view is not a duplicate)", len(j.entries))
	}
	j.push(jumpEntry{view: viewTree, itemID: "T1"})
	if len(j.entries) != 3 {
		t.Fatalf("len = %d, want 3 (dedup only checks the immediately-preceding entry)", len(j.entries))
	}
}

func TestJumplistPushCapsDroppingOldest(t *testing.T) {
	t.Parallel()
	const pushed = jumplistCap + 10
	var j jumplist
	for i := 0; i < pushed; i++ {
		j.push(jumpEntry{view: viewTree, itemID: strconv.Itoa(i)})
	}
	if len(j.entries) != jumplistCap {
		t.Fatalf("len = %d, want %d", len(j.entries), jumplistCap)
	}
	if got, want := j.entries[0].itemID, strconv.Itoa(pushed-jumplistCap); got != want {
		t.Fatalf("oldest surviving entry = %q, want %q (first %d pushes evicted)", got, want, pushed-jumplistCap)
	}
	if got, want := j.entries[len(j.entries)-1].itemID, strconv.Itoa(pushed-1); got != want {
		t.Fatalf("newest entry = %q, want %q", got, want)
	}
	if j.pos != jumplistCap {
		t.Fatalf("pos = %d, want %d", j.pos, jumplistCap)
	}
}

func TestJumplistPushAfterBackTruncatesForwardHistory(t *testing.T) {
	t.Parallel()
	var j jumplist
	j.push(jumpEntry{view: viewTree, itemID: "A"})
	j.push(jumpEntry{view: viewBoard, itemID: "B"})
	j.push(jumpEntry{view: viewStats, itemID: "C"})

	if _, ok := j.back(); !ok {
		t.Fatal("back() should succeed with history present")
	}
	j.push(jumpEntry{view: viewTree, itemID: "D"})

	want := []string{"A", "B", "D"}
	if len(j.entries) != len(want) {
		t.Fatalf("entries = %+v, want itemIDs %v", j.entries, want)
	}
	for i, id := range want {
		if j.entries[i].itemID != id {
			t.Fatalf("entries[%d].itemID = %q, want %q", i, j.entries[i].itemID, id)
		}
	}
	if j.pos != len(j.entries) {
		t.Fatalf("pos = %d, want %d (pushed entry becomes the new head)", j.pos, len(j.entries))
	}
}

func TestJumplistBackForwardBounds(t *testing.T) {
	t.Parallel()
	var j jumplist
	if _, ok := j.back(); ok {
		t.Fatal("back() on an empty jumplist must return ok=false")
	}
	if _, ok := j.forward(); ok {
		t.Fatal("forward() on an empty jumplist must return ok=false")
	}

	j.push(jumpEntry{view: viewTree, itemID: "A"})
	j.push(jumpEntry{view: viewBoard, itemID: "B"})

	if _, ok := j.forward(); ok {
		t.Fatal("forward() right after a push must return ok=false (already at the head)")
	}
	e, ok := j.back()
	if !ok || e.itemID != "B" {
		t.Fatalf("first back() = %+v, %v, want {itemID:B}, true", e, ok)
	}
	e, ok = j.back()
	if !ok || e.itemID != "A" {
		t.Fatalf("second back() = %+v, %v, want {itemID:A}, true", e, ok)
	}
	if _, ok := j.back(); ok {
		t.Fatal("back() past the oldest entry must return ok=false")
	}
	e, ok = j.forward()
	if !ok || e.itemID != "B" {
		t.Fatalf("forward() = %+v, %v, want {itemID:B}, true", e, ok)
	}
	if _, ok := j.forward(); ok {
		t.Fatal("forward() at the newest entry must return ok=false")
	}
}

func TestSwitchViewToCurrentViewPushesNoJump(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	updated, _ := m.Update(key("1"))
	m2 := updated.(model)
	if len(m2.jumps.entries) != 0 {
		t.Fatalf("re-selecting the current view must not record a jump, got %+v", m2.jumps.entries)
	}
	updated, _ = m2.Update(key("2"))
	m3 := updated.(model)
	if len(m3.jumps.entries) != 1 {
		t.Fatalf("a real view switch must record exactly one jump, got %+v", m3.jumps.entries)
	}
}

func TestJumplistRoutingViaCtrlOAndCtrlCloseBracket(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	m.jumps.push(jumpEntry{view: viewTree, itemID: "T1"})
	m.jumps.push(jumpEntry{view: viewBoard, itemID: ""})
	m.view = viewStats

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m2 := updated.(model)
	if m2.view != viewBoard {
		t.Fatalf("first ctrl+o should jump back to the most recently recorded view, got %v", m2.view)
	}

	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	m3 := updated.(model)
	if m3.view != viewTree {
		t.Fatalf("second ctrl+o should jump back to the earlier recorded view, got %v", m3.view)
	}
	ts := m3.screens[viewTree].(*treeScreen)
	if got := ts.tree.selectedID(); got != "T1" {
		t.Fatalf("ctrl+o should focus the jumped-to item, selectedID = %q, want %q", got, "T1")
	}

	updated, _ = m3.Update(tea.KeyMsg{Type: tea.KeyCtrlCloseBracket})
	m4 := updated.(model)
	if m4.view != viewBoard {
		t.Fatalf("ctrl+] should move forward to the view it came from, got %v", m4.view)
	}
}
