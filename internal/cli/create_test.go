package cli

import (
	"os"
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

func restoreArgs(t *testing.T, argv []string) {
	t.Helper()
	saved := os.Args
	t.Cleanup(func() { os.Args = saved })
	os.Args = argv
}
