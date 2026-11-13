package config_test

import (
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

const boardBaseConfig = `version: 2

project:
  key: KIRA
  name: kira

workflows:
  ticket:
    states:
      - { key: TODO,        category: todo }
      - { key: IN_PROGRESS, category: doing, wip: 3 }   # advisory column cap
      - { key: DONE,        category: done }
    initial: TODO

filters: {}                   # named saved queries

commit:
  mode: auto                  # auto | manual
`

func addBoard(t *testing.T, data string, b datamodel.Board) string {
	t.Helper()
	out, err := config.AddBoard([]byte(data), b)
	if err != nil {
		t.Fatalf("AddBoard(%s): %v", b.Key, err)
	}
	return string(out)
}

func withKIRABoard(t *testing.T) string {
	t.Helper()
	return addBoard(t, boardBaseConfig, datamodel.Board{Key: "KIRA", Name: "kira", Default: true})
}

func TestAddBoardAppendsAfterExistingBoard(t *testing.T) {
	out := addBoard(t, withKIRABoard(t), datamodel.Board{Key: "XYZ", Name: "Beta"})

	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if len(cfg.Boards) != 2 {
		t.Fatalf("parsed boards = %+v, want 2 entries", cfg.Boards)
	}
	if cfg.Boards[0].Key != "KIRA" || !cfg.Boards[0].Default {
		t.Errorf("first board = %+v, want KIRA default", cfg.Boards[0])
	}
	if cfg.Boards[1].Key != "XYZ" || cfg.Boards[1].Default {
		t.Errorf("second board = %+v, want XYZ non-default", cfg.Boards[1])
	}
}

func TestAddBoardOpensMissingBoardsList(t *testing.T) {
	out := addBoard(t, boardBaseConfig, datamodel.Board{Key: "KIRA", Name: "kira", Default: true})

	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if len(cfg.Boards) != 1 || cfg.Boards[0].Key != "KIRA" || !cfg.Boards[0].Default {
		t.Errorf("parsed boards = %+v, want single KIRA default", cfg.Boards)
	}
}

func TestAddBoardSecondAppendKeepsEarlierBoards(t *testing.T) {
	first := addBoard(t, withKIRABoard(t), datamodel.Board{Key: "XYZ", Name: "Beta"})
	out := addBoard(t, first, datamodel.Board{Key: "ABC", Name: "Gamma"})

	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if len(cfg.Boards) != 3 || cfg.Boards[2].Key != "ABC" {
		t.Errorf("parsed boards = %+v, want 3 with ABC last", cfg.Boards)
	}
}

func TestUpdateBoardRenamePreservesOtherEntries(t *testing.T) {
	base := addBoard(t, withKIRABoard(t), datamodel.Board{Key: "XYZ", Name: "Beta"})
	out, err := config.UpdateBoard([]byte(base), "xyz", func(b datamodel.Board) datamodel.Board {
		b.Name = "Beta Squad"
		return b
	})
	if err != nil {
		t.Fatalf("UpdateBoard rename: %v", err)
	}
	cfg, err := config.Parse(out)
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if b, ok := cfg.BoardByKey("XYZ"); !ok || b.Name != "Beta Squad" {
		t.Fatalf("rename not applied: %+v", cfg.Boards)
	}
	if !strings.Contains(string(out), "{ key: KIRA, name: kira, default: true }") {
		t.Errorf("unrelated board entry changed:\n%s", out)
	}
}

func TestUpdateBoardArchiveSetsFlag(t *testing.T) {
	base := addBoard(t, withKIRABoard(t), datamodel.Board{Key: "XYZ", Name: "Beta"})
	out, err := config.UpdateBoard([]byte(base), "XYZ", func(b datamodel.Board) datamodel.Board {
		b.Archived = true
		return b
	})
	if err != nil {
		t.Fatalf("UpdateBoard archive: %v", err)
	}
	cfg, _ := config.Parse(out)
	if b, _ := cfg.BoardByKey("XYZ"); !b.Archived {
		t.Fatalf("archive flag not set: %+v", cfg.Boards)
	}
}

func TestUpdateBoardUnknownKeyErrors(t *testing.T) {
	base := addBoard(t, withKIRABoard(t), datamodel.Board{Key: "XYZ", Name: "Beta"})
	if _, err := config.UpdateBoard([]byte(base), "NOPE", func(b datamodel.Board) datamodel.Board { return b }); err == nil {
		t.Fatal("expected an error updating an unknown board key")
	}
}

func TestAddBoardQuotesFlowBreakingValues(t *testing.T) {
	for _, name := range []string{"Ops,Live", "a{b}c", "x[y]z", "key: val"} {
		out := addBoard(t, withKIRABoard(t), datamodel.Board{Key: "XYZ", Name: name})
		cfg, err := config.Parse([]byte(out))
		if err != nil {
			t.Fatalf("parse after quoting %q: %v\n%s", name, err, out)
		}
		b, ok := cfg.BoardByKey("XYZ")
		if !ok || b.Name != name {
			t.Fatalf("name %q did not round-trip: %+v", name, cfg.Boards)
		}
	}
}

func TestAddBoardAppendsToFlowList(t *testing.T) {
	flowCfg := boardBaseConfig +
		"boards: [{ key: KIRA, name: kira, default: true }]\n"
	out := addBoard(t, flowCfg, datamodel.Board{Key: "XYZ", Name: "Beta"})
	cfg, err := config.Parse([]byte(out))
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, out)
	}
	if len(cfg.Boards) != 2 || cfg.Boards[1].Key != "XYZ" {
		t.Fatalf("flow-list append: %+v", cfg.Boards)
	}
}

func TestUpdateBoardPreservesTrailingComment(t *testing.T) {
	base := addBoard(t, withKIRABoard(t), datamodel.Board{Key: "XYZ", Name: "Beta"})
	base = strings.Replace(base, "{ key: XYZ, name: Beta }", "{ key: XYZ, name: Beta }  # squad", 1)
	out, err := config.UpdateBoard([]byte(base), "XYZ", func(b datamodel.Board) datamodel.Board {
		b.Name = "Beta Squad"
		return b
	})
	if err != nil {
		t.Fatalf("UpdateBoard: %v", err)
	}
	if !strings.Contains(string(out), "# squad") {
		t.Errorf("trailing comment dropped:\n%s", out)
	}
	cfg, _ := config.Parse(out)
	if b, _ := cfg.BoardByKey("XYZ"); b.Name != "Beta Squad" {
		t.Fatalf("rename not applied: %+v", cfg.Boards)
	}
}

func TestUpdateBoardPreservesCommentContainingBrace(t *testing.T) {
	base := addBoard(t, withKIRABoard(t), datamodel.Board{Key: "XYZ", Name: "Beta"})
	base = strings.Replace(base, "{ key: XYZ, name: Beta }", "{ key: XYZ, name: Beta }  # squad {ops}", 1)
	out, err := config.UpdateBoard([]byte(base), "XYZ", func(b datamodel.Board) datamodel.Board {
		b.Name = "Beta Squad"
		return b
	})
	if err != nil {
		t.Fatalf("UpdateBoard: %v", err)
	}
	if !strings.Contains(string(out), "# squad {ops}") {
		t.Errorf("comment containing '}' mangled:\n%s", out)
	}
	cfg, err := config.Parse(out)
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if b, _ := cfg.BoardByKey("XYZ"); b.Name != "Beta Squad" {
		t.Fatalf("rename not applied: %+v", cfg.Boards)
	}
}

func TestUpdateBoardQuotedNameWithBraceRoundTrips(t *testing.T) {
	base := addBoard(t, withKIRABoard(t), datamodel.Board{Key: "XYZ", Name: "Ops}Live"})
	out, err := config.UpdateBoard([]byte(base), "XYZ", func(b datamodel.Board) datamodel.Board {
		b.Archived = true
		return b
	})
	if err != nil {
		t.Fatalf("UpdateBoard: %v", err)
	}
	cfg, err := config.Parse(out)
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if b, _ := cfg.BoardByKey("XYZ"); b.Name != "Ops}Live" || !b.Archived {
		t.Fatalf("quoted-brace entry did not round-trip: %+v", cfg.Boards)
	}
}

func TestUpdateBoardRefusesMultiLineEntry(t *testing.T) {
	base := boardBaseConfig +
		"boards:\n  - key: XYZ\n    name: Beta\n"
	_, err := config.UpdateBoard([]byte(base), "XYZ", func(b datamodel.Board) datamodel.Board {
		b.Name = "Beta Squad"
		return b
	})
	if err == nil || !strings.Contains(err.Error(), "multi-line entry") {
		t.Fatalf("error = %v, want multi-line refusal", err)
	}
}

func TestUpdateBoardInsideInlineFlowList(t *testing.T) {
	base := boardBaseConfig +
		"boards: [{ key: KIRA, name: kira, default: true }, { key: XYZ, name: Beta }]\n"
	out, err := config.UpdateBoard([]byte(base), "XYZ", func(b datamodel.Board) datamodel.Board {
		b.Name = "Beta Squad"
		return b
	})
	if err != nil {
		t.Fatalf("UpdateBoard: %v", err)
	}
	cfg, err := config.Parse(out)
	if err != nil {
		t.Fatalf("result does not parse: %v", err)
	}
	if b, _ := cfg.BoardByKey("XYZ"); b.Name != "Beta Squad" {
		t.Fatalf("flow-list update not applied: %+v", cfg.Boards)
	}
	if b, _ := cfg.BoardByKey("KIRA"); !b.Default {
		t.Fatalf("sibling flow entry damaged: %+v", cfg.Boards)
	}
}

func TestUpdateBoardNonASCIISiblingNameByteOffset(t *testing.T) {
	base := boardBaseConfig +
		"boards: [{ key: KIRA, name: Zürich, default: true }, { key: XYZ, name: Beta }]\n"
	out, err := config.UpdateBoard([]byte(base), "XYZ", func(b datamodel.Board) datamodel.Board {
		b.Name = "Beta Squad"
		return b
	})
	if err != nil {
		t.Fatalf("UpdateBoard: %v", err)
	}
	cfg, err := config.Parse(out)
	if err != nil {
		t.Fatalf("result does not parse: %v\n%s", err, out)
	}
	if b, ok := cfg.BoardByKey("XYZ"); !ok || b.Name != "Beta Squad" {
		t.Fatalf("rename not applied: %+v", cfg.Boards)
	}
	if b, ok := cfg.BoardByKey("KIRA"); !ok || b.Name != "Zürich" {
		t.Fatalf("multi-byte sibling entry damaged: %+v", cfg.Boards)
	}
}

func TestAddBoardPreservesUnrelatedLines(t *testing.T) {
	out := addBoard(t, withKIRABoard(t), datamodel.Board{Key: "XYZ", Name: "Beta"})

	for line := range strings.SplitSeq(boardBaseConfig, "\n") {
		if strings.HasPrefix(line, "version:") {
			continue
		}
		if line != "" && !strings.Contains(out, line) {
			t.Errorf("unrelated line dropped: %q\n--- got ---\n%s", line, out)
		}
	}
}
