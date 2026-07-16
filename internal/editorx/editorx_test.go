package editorx

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCommandPrecedence(t *testing.T) {
	cases := []struct {
		name       string
		configured string
		editor     string
		visual     string
		want       string
		wantErr    bool
	}{
		{"configured beats env", "cfg-editor --flag", "env-editor", "vis-editor", "cfg-editor --flag", false},
		{"blank configured falls back", "   ", "env-editor", "vis-editor", "vis-editor", false},
		{"VISUAL beats EDITOR", "", "env-editor", "vis-editor", "vis-editor", false},
		{"EDITOR fallback", "", "env-editor", "", "env-editor", false},
		{"nothing set", "", "", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("EDITOR", tc.editor)
			t.Setenv("VISUAL", tc.visual)
			got, err := Command(tc.configured)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Command(%q) = %v, want error", tc.configured, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Command(%q): %v", tc.configured, err)
			}
			if got.raw != tc.want {
				t.Errorf("Command(%q) = %q, want %q", tc.configured, got.raw, tc.want)
			}
		})
	}
}

func TestViewAppendsReadonlyFlagForVimFamily(t *testing.T) {
	cases := []struct {
		name   string
		editor string
		want   []string
	}{
		{"vi", "vi", []string{"sh", "-c", `vi "$@"`, "vi", "-R", "/x.md"}},
		{"vim", "vim", []string{"sh", "-c", `vim "$@"`, "vim", "-R", "/x.md"}},
		{"nvim with path", "/usr/bin/nvim", []string{"sh", "-c", `/usr/bin/nvim "$@"`, "/usr/bin/nvim", "-R", "/x.md"}},
		{"gvim", "gvim", []string{"sh", "-c", `gvim "$@"`, "gvim", "-R", "/x.md"}},
		{"vim with flags", "vim -u NONE", []string{"sh", "-c", `vim -u NONE "$@"`, "vim -u NONE", "-R", "/x.md"}},
		{"non-vim editor", "nano", []string{"sh", "-c", `nano "$@"`, "nano", "/x.md"}},
		{"vim-adjacent name", "vimdiff", []string{"sh", "-c", `vimdiff "$@"`, "vimdiff", "/x.md"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ed, err := Command(tc.editor)
			if err != nil {
				t.Fatalf("Command(%q): %v", tc.editor, err)
			}
			cmd := ed.View("/x.md")
			if !slices.Equal(cmd.Args, tc.want) {
				t.Errorf("View(%q) args = %v, want %v", tc.editor, cmd.Args, tc.want)
			}
		})
	}
}

func TestEditPreservesQuotedEditorPath(t *testing.T) {
	dir := t.TempDir()
	editorPath := filepath.Join(dir, "my editor.sh")
	argsFile := filepath.Join(dir, "args.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + argsFile + "\"\n"
	if err := os.WriteFile(editorPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	configured := `"` + editorPath + `" --wait`

	ed, err := Command(configured)
	if err != nil {
		t.Fatalf("Command(%q): %v", configured, err)
	}
	if err := ed.Edit("/draft.md", Stdio{}); err != nil {
		t.Fatalf("Edit(%q): %v", configured, err)
	}
	got, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("editor script did not run: %v", err)
	}
	if want := "--wait\n/draft.md\n"; string(got) != want {
		t.Errorf("editor argv = %q, want %q", got, want)
	}
}

func TestEditReportsEditorFailure(t *testing.T) {
	ed, err := Command("false")
	if err != nil {
		t.Fatalf("Command: %v", err)
	}
	err = ed.Edit("/draft.md", Stdio{Out: &strings.Builder{}, Err: &strings.Builder{}})
	if err == nil || !strings.Contains(err.Error(), "editor false") {
		t.Fatalf("Edit with failing editor = %v, want editor false error", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Edit error = %v, want it to wrap an *exec.ExitError (errors.As)", err)
	}
}
