package cli

import (
	"os"
	"slices"
	"testing"
)

func TestChdirArg(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want string
	}{
		{"absent", []string{"kira", "create", "ticket"}, ""},
		{"space form", []string{"kira", "create", "-C", "/repo", "ticket"}, "/repo"},
		{"long space form", []string{"kira", "create", "--C", "/repo", "ticket"}, "/repo"},
		{"attached shorthand", []string{"kira", "create", "-C/repo", "ticket"}, "/repo"},
		{"long equals form", []string{"kira", "create", "--C=/repo", "ticket"}, "/repo"},
		{"stops at --", []string{"kira", "create", "--", "-C", "/repo"}, ""},
		{"preceded by a known bool flag", []string{"kira", "--quiet", "-C", "/repo", "create"}, "/repo"},
		{"does not hijack another flag's literal -C value", []string{"kira", "create", "ticket", "--title=-C"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			restoreArgs(t, c.argv)
			if got := chdirArg(); got != c.want {
				t.Errorf("chdirArg() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestStripGlobalPrefix(t *testing.T) {
	cases := []struct {
		name      string
		argv      []string
		args      []string
		wantChdir string
		wantRest  []string
	}{
		{"chdir before find", []string{"--no-color", "-C", "/repo", "find", "pattern", "--json"},
			[]string{"--no-color", "-C", "/repo", "pattern", "--json"},
			"/repo", []string{"pattern", "--json"}},
		{"no chdir", []string{"find", "pattern"}, []string{"pattern"}, "", []string{"pattern"}},
		{"ripgrep's own -C after find is not chdir", []string{"find", "pattern", "-C", "3"},
			[]string{"pattern", "-C", "3"}, "", []string{"pattern", "-C", "3"}},
		{"bridged: find absent from argv, rest passes through", []string{"tui"},
			[]string{"--no-color", "pattern"}, "", []string{"--no-color", "pattern"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			chdir, rest := stripGlobalPrefix(c.argv, c.args, "find")
			if chdir != c.wantChdir {
				t.Errorf("chdir = %q, want %q", chdir, c.wantChdir)
			}
			if !slices.Equal(rest, c.wantRest) {
				t.Errorf("rest = %q, want %q", rest, c.wantRest)
			}
		})
	}
}

func restoreArgs(t *testing.T, argv []string) {
	t.Helper()
	saved := os.Args
	t.Cleanup(func() { os.Args = saved })
	os.Args = argv
}
