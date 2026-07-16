package tui

import (
	"os"
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestViewProcessOpensReadonlyTempCopy(t *testing.T) {
	t.Parallel()
	s, cfg, _ := initRepo(t)
	num := createTicket(t, s, cfg, "Viewable")
	cfg.UI.Editor = "vi"
	srcPath, srcContent, err := s.ResolveItemFile(cfg, num)
	if err != nil {
		t.Fatalf("ResolveItemFile: %v", err)
	}

	cmd, tmpPath, err := viewProcess(s, cfg, num)
	if err != nil {
		t.Fatalf("viewProcess: %v", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	if tmpPath == srcPath {
		t.Fatal("viewer must open a temp copy, not the live store file")
	}
	if want := []string{"sh", "-c", `vi "$@"`, "vi", "-R", tmpPath}; !slices.Equal(cmd.Args, want) {
		t.Errorf("viewer args = %v, want %v", cmd.Args, want)
	}
	fi, err := os.Stat(tmpPath)
	if err != nil {
		t.Fatalf("stat temp copy: %v", err)
	}
	if fi.Mode().Perm() != 0o400 {
		t.Errorf("temp copy mode = %v, want 0400", fi.Mode().Perm())
	}
	copied, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("read temp copy: %v", err)
	}
	if string(copied) != srcContent {
		t.Error("temp copy content differs from the source item")
	}
	after, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if string(after) != srcContent {
		t.Error("source item changed while preparing the view")
	}
}

func TestTreeEditWithoutEditorShowsBarError(t *testing.T) {
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	s, cfg, _ := initRepo(t)
	createTicket(t, s, cfg, "Editable")
	data, err := loadFilteredTree(s, cfg, "")
	if err != nil {
		t.Fatalf("loadFilteredTree: %v", err)
	}
	m := newModel(s, cfg, asciiTheme(), iconSet{mode: datamodel.IconText}, false)
	m.width, m.height = 100, 12
	ts := m.screens[viewTree].(*treeScreen)
	ts.setData(&m, data)

	if cmd := ts.update(&m, "e"); cmd != nil {
		t.Error("e without any editor must not suspend the TUI")
	}
	if !m.bar.msgErr || m.bar.msg == "" {
		t.Errorf("missing editor must land in the bar, got msg=%q err=%v", m.bar.msg, m.bar.msgErr)
	}
}

func TestEditorDoneMsgErrorLandsInBarAndResetsCache(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	ts := m.screens[viewTree].(*treeScreen)
	ts.host.cache["E1"] = sampleDetail()

	updated, cmd := m.Update(editorDoneMsg{err: errStub("boom")})
	got := updated.(model)
	if got.bar.msg != "boom" || !got.bar.msgErr {
		t.Errorf("bar = (%q, %v), want the editor error", got.bar.msg, got.bar.msgErr)
	}
	if len(ts.host.cache) != 0 {
		t.Error("editorDoneMsg must reset the detail cache")
	}
	if cmd == nil {
		t.Error("editorDoneMsg must request a data refresh")
	}
}

func TestEditorDoneMsgSuccessRequestsQuietRefresh(t *testing.T) {
	t.Parallel()
	m := newTestModel(100, 12, true)
	ts := m.screens[viewTree].(*treeScreen)
	ts.host.cache["E1"] = sampleDetail()

	updated, cmd := m.Update(editorDoneMsg{})
	got := updated.(model)
	if got.bar.msg != "" || got.bar.msgErr {
		t.Errorf("bar = (%q, %v), want it untouched on success", got.bar.msg, got.bar.msgErr)
	}
	if len(ts.host.cache) != 0 {
		t.Error("editorDoneMsg must reset the detail cache")
	}
	if cmd == nil {
		t.Error("editorDoneMsg must request a data refresh")
	}
}
