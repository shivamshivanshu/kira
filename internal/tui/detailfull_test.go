package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func sampleDetail() *datamodel.ShowResult {
	from := "TODO"
	return &datamodel.ShowResult{
		Number: "KIRA-142", Title: "Fix race in order-book snapshot merge",
		State: "IN_PROGRESS", Category: "doing", Owner: strptr("shivam"),
		Priority: strptr("P1"), Labels: []string{"bug", "orderbook"},
		Body: "The snapshot merge path drops updates when two feed threads race.\n\n" +
			"## Acceptance criteria\n\n- TSan clean on order_book_test\n- No p99 regression\n",
		Comments: []datamodel.CommentView{
			{Author: "shivam", Ts: "2026-07-11 18:30", Text: "Confirmed repro with TSan; missing acquire fence."},
		},
		LinkedCommits: []datamodel.CommitLink{
			{SHA: "a1b2c3d4e5", Subject: "fix acquire fence on the consumer side", Author: "shivam", Ts: "2026-07-11T18:31:00+05:30"},
			{SHA: "f6a7b8c9d0", Subject: "add burst regression bench", Author: "shivam", Ts: "2026-07-11T19:02:00+05:30"},
		},
		HistoryTail: []datamodel.HistoryEvent{
			{Ts: "2026-07-11T18:30:00+05:30", Field: "state", From: &from, To: strptr("IN_PROGRESS")},
			{Ts: "2026-07-10T09:00:00+05:30", Field: "owner", From: nil, To: strptr("shivam")},
		},
	}
}

func TestDetailPanelFull(t *testing.T) {
	got := newDetailPanel().render(asciiTheme(), sampleDetail(), 100, 40)
	golden.RequireEqual(t, []byte(got))
}

func TestDetailPanelNarrowGuard(t *testing.T) {
	got := newDetailPanel().render(asciiTheme(), sampleDetail(), 20, 40)
	golden.RequireEqual(t, []byte(got))
}

func TestDetailPanelRendersSubtypeWhenSet(t *testing.T) {
	res := sampleDetail()
	res.Subtype = strptr("bug")
	got := newDetailPanel().render(asciiTheme(), res, 100, 40)
	if !strings.Contains(got, "subtype bug") {
		t.Fatalf("detail should render the subtype:\n%s", got)
	}
	if strings.Contains(newDetailPanel().render(asciiTheme(), sampleDetail(), 100, 40), "subtype") {
		t.Fatal("detail should omit subtype when unset")
	}
}

func TestDetailPanelNil(t *testing.T) {
	if got := newDetailPanel().render(asciiTheme(), nil, 100, 40); got == "" {
		t.Fatal("nil result should render a placeholder")
	}
}

func TestDetailPanelCommitSelection(t *testing.T) {
	res := sampleDetail()
	d := newDetailPanel()
	d.update(nil, res, "]")
	if d.commitSel != 1 {
		t.Fatalf("commitSel after ] = %d, want 1", d.commitSel)
	}
	d.update(nil, res, "]")
	if d.commitSel != 1 {
		t.Fatalf("commitSel clamped at last = %d, want 1", d.commitSel)
	}
	d.update(nil, res, "[")
	d.update(nil, res, "[")
	if d.commitSel != 0 {
		t.Fatalf("commitSel clamped at first = %d, want 0", d.commitSel)
	}
	if sha := selectedCommit(res, d.commitSel); sha != "a1b2c3d4e5" {
		t.Fatalf("selectedCommit = %q", sha)
	}
}

func TestDetailPanelEnterNoStoreNoCmd(t *testing.T) {
	if cmd := newDetailPanel().update(&model{}, sampleDetail(), "enter"); cmd != nil {
		t.Fatal("enter with nil store must not issue a command")
	}
}

func TestTreeDetailPanelShowsLinkedData(t *testing.T) {
	m := newTestModel(100, 40, true)
	ts := m.screens[viewTree].(*treeScreen)
	ts.host.cache["E1"] = sampleDetail()
	ts.focus = paneDetail
	ts.syncDetail(&m)
	got := ts.view(&m, 100, 40)
	for _, want := range []string{"Linked commits", "History", "a1b2c3d", "state: TODO -> IN_PROGRESS"} {
		if !strings.Contains(got, want) {
			t.Fatalf("tree detail pane should render %q; got:\n%s", want, got)
		}
	}
}

func TestDetailCacheKeyedByResultPointer(t *testing.T) {
	res := sampleDetail()
	d := newDetailPanel()
	first := strings.Join(d.contentLines(asciiTheme(), res, 100), "\n")

	res.Title = "mutated after first render"
	again := strings.Join(d.contentLines(asciiTheme(), res, 100), "\n")
	if again != first {
		t.Fatal("same (res,width) key must serve the cached lines, ignoring an in-place mutation of the same result")
	}

	other := sampleDetail()
	other.Title = "a distinctly different ticket"
	fresh := strings.Join(d.contentLines(asciiTheme(), other, 100), "\n")
	if fresh == first {
		t.Fatal("a fresh result pointer must invalidate the cache and rebuild, not reuse the first render")
	}
	if !strings.Contains(fresh, other.Title) {
		t.Fatalf("rebuilt content must reflect the new result's data, got:\n%s", fresh)
	}
}

func TestDetailPanelScrollClamp(t *testing.T) {
	d := newDetailPanel()
	for i := 0; i < 50; i++ {
		d.update(nil, sampleDetail(), "j")
	}
	d.render(asciiTheme(), sampleDetail(), 100, 8)
	lines := d.contentLines(asciiTheme(), sampleDetail(), 100)
	if d.scroll > max(0, len(lines)-8) {
		t.Fatalf("scroll %d exceeds max %d", d.scroll, max(0, len(lines)-8))
	}
}
