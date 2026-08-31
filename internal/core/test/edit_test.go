package core_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

func writeEditorScript(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "editor.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "sh "+path)
	return path
}

func bumpUpdated(content string) string {
	lines := strings.Split(content, "\n")
	for i, ln := range lines {
		if strings.HasPrefix(ln, datamodel.KeyUpdated+": ") {
			lines[i] = datamodel.KeyUpdated + ": 2099-01-01T00:00:00Z"
		}
	}
	return strings.Join(lines, "\n")
}

func TestEditorModeSurfacesParseError(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "keepme")
	before := mustReadItem(t, s, res.ID)

	counter := filepath.Join(t.TempDir(), "counter")
	t.Setenv("KIRA_COUNTER", counter)
	writeEditorScript(t,
		"n=$(cat \"$KIRA_COUNTER\" 2>/dev/null || echo 0)\n"+
			"n=$((n+1)); echo \"$n\" > \"$KIRA_COUNTER\"\n"+
			"if [ \"$n\" -eq 1 ]; then printf 'not a valid item\\n' > \"$1\"; fi\n")

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{})
	if err == nil {
		t.Fatal("expected a parse/validation error, got nil")
	}
	var e *errx.Error
	if !errors.As(err, &e) || e.Code != errx.ExitUser {
		t.Fatalf("want ExitUser parse error, got %v (%T)", err, err)
	}
	if got := mustReadItem(t, s, res.ID); got != before {
		t.Fatalf("a refused parse must not mutate the file:\n%s", got)
	}
}

func TestEditorModeRefusesLostUpdate(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "original")

	orig := mustReadItem(t, s, res.ID)
	concurrent := writeTempItem(t, bumpUpdated(orig))
	buffer := writeTempItem(t, strings.Replace(orig, "title: \"original\"", "title: \"edited by user\"", 1))

	t.Setenv("KIRA_ITEM", storage.New(s.Root()).ItemPath(res.ID))
	t.Setenv("KIRA_CONCURRENT", concurrent)
	t.Setenv("KIRA_BUFFER", buffer)
	writeEditorScript(t, "cp \"$KIRA_CONCURRENT\" \"$KIRA_ITEM\"\ncp \"$KIRA_BUFFER\" \"$1\"\n")

	_, err := s.Edit(cfg, "KIRA-1", core.EditOpts{})
	if err == nil {
		t.Fatal("expected a lost-update conflict, got nil")
	}
	var e *errx.Error
	if !errors.As(err, &e) {
		t.Fatalf("want *errx.Error, got %T: %v", err, err)
	}
	if e.Code != errx.ExitConflict {
		t.Fatalf("exit code = %d, want %d (ExitConflict)", e.Code, errx.ExitConflict)
	}
	if !strings.Contains(e.Hint, "re-run") {
		t.Fatalf("conflict missing retry hint: %q", e.Hint)
	}
	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, "title: \"original\"") {
		t.Fatalf("lost-update refusal must not overwrite the concurrent version:\n%s", got)
	}
}

func TestEditorModeRefusesLostComment(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "original")

	orig := mustReadItem(t, s, res.ID)
	commented, err := codec.AppendCommentToDocument(orig, datamodel.Comment{
		ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", Author: "alice", Ts: "2026-01-01T00:00:00Z", Body: "important",
	})
	if err != nil {
		t.Fatalf("AppendCommentToDocument: %v", err)
	}
	concurrent := writeTempItem(t, commented)
	buffer := writeTempItem(t, strings.Replace(orig, "title: \"original\"", "title: \"edited by user\"", 1))

	t.Setenv("KIRA_ITEM", storage.New(s.Root()).ItemPath(res.ID))
	t.Setenv("KIRA_CONCURRENT", concurrent)
	t.Setenv("KIRA_BUFFER", buffer)
	writeEditorScript(t, "cp \"$KIRA_CONCURRENT\" \"$KIRA_ITEM\"\ncp \"$KIRA_BUFFER\" \"$1\"\n")

	_, err = s.Edit(cfg, "KIRA-1", core.EditOpts{})
	var e *errx.Error
	if !errors.As(err, &e) || e.Code != errx.ExitConflict {
		t.Fatalf("want ExitConflict for a comment appended mid-session, got %v (%T)", err, err)
	}
	if got := mustReadItem(t, s, res.ID); !strings.Contains(got, "important") {
		t.Fatalf("refused edit must preserve the concurrent comment:\n%s", got)
	}
}

func TestFieldEditAfterCommentDoesNotConflict(t *testing.T) {
	s, cfg := newStore(t)
	res := mustCreate(t, s, cfg, "original")
	if _, err := s.Comment(cfg, "KIRA-1", core.CommentOpts{Message: "important", HasMessage: true}); err != nil {
		t.Fatalf("Comment: %v", err)
	}

	fields := []core.FieldEdit{{Key: "title", Value: "edited"}}
	if _, err := s.Edit(cfg, "KIRA-1", core.EditOpts{Fields: fields}); err != nil {
		t.Fatalf("--field edit after a comment must not conflict: %v", err)
	}
	got := mustReadItem(t, s, res.ID)
	if !strings.Contains(got, "title: \"edited\"") || !strings.Contains(got, "important") {
		t.Fatalf("--field edit lost the title change or the comment:\n%s", got)
	}
}
