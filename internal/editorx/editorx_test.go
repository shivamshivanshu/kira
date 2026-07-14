package editorx

import (
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
		{"blank configured falls back", "   ", "env-editor", "vis-editor", "env-editor", false},
		{"EDITOR beats VISUAL", "", "env-editor", "vis-editor", "env-editor", false},
		{"VISUAL fallback", "", "", "vis-editor", "vis-editor", false},
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
			if joined := strings.Join(got, " "); joined != tc.want {
				t.Errorf("Command(%q) = %q, want %q", tc.configured, joined, tc.want)
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
		{"vi", "vi", []string{"vi", "-R", "/x.md"}},
		{"vim", "vim", []string{"vim", "-R", "/x.md"}},
		{"nvim with path", "/usr/bin/nvim", []string{"/usr/bin/nvim", "-R", "/x.md"}},
		{"gvim", "gvim", []string{"gvim", "-R", "/x.md"}},
		{"vim with flags", "vim -u NONE", []string{"vim", "-u", "NONE", "-R", "/x.md"}},
		{"non-vim editor", "nano", []string{"nano", "/x.md"}},
		{"vim-adjacent name", "vimdiff", []string{"vimdiff", "/x.md"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := View(tc.editor, "/x.md")
			if err != nil {
				t.Fatalf("View(%q): %v", tc.editor, err)
			}
			if !slices.Equal(cmd.Args, tc.want) {
				t.Errorf("View(%q) args = %v, want %v", tc.editor, cmd.Args, tc.want)
			}
		})
	}
}
