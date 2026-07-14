package tui

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestResolveIconMode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		mode datamodel.IconMode
		env  map[string]string
		tty  bool
		want datamodel.IconMode
	}{
		{"config nerd is absolute even piped", datamodel.IconNerd, map[string]string{"KIRA_ICONS": "text", "TERM": "dumb"}, false, datamodel.IconNerd},
		{"config emoji wins over env", datamodel.IconEmoji, map[string]string{"KIRA_ICONS": "nerd"}, true, datamodel.IconEmoji},
		{"config text wins over env", datamodel.IconText, map[string]string{"KIRA_ICONS": "nerd"}, true, datamodel.IconText},
		{"legacy always maps to nerd, absolute even piped", datamodel.IconAlways, nil, false, datamodel.IconNerd},
		{"legacy never maps to text", datamodel.IconNever, nil, true, datamodel.IconText},
		{"auto env nerd is absolute even piped", datamodel.IconAuto, map[string]string{"KIRA_ICONS": "nerd"}, false, datamodel.IconNerd},
		{"auto consults env emoji", datamodel.IconAuto, map[string]string{"KIRA_ICONS": "emoji"}, true, datamodel.IconEmoji},
		{"auto consults env legacy always", datamodel.IconAuto, map[string]string{"KIRA_ICONS": "always"}, true, datamodel.IconNerd},
		{"auto consults env legacy never", datamodel.IconAuto, map[string]string{"KIRA_ICONS": "never"}, true, datamodel.IconText},
		{"auto on a tty defaults to nerd with utf-8 locale", datamodel.IconAuto, map[string]string{"LANG": "en_US.UTF-8"}, true, datamodel.IconNerd},
		{"auto on a tty defaults to nerd without locale", datamodel.IconAuto, nil, true, datamodel.IconNerd},
		{"auto piped degrades to text", datamodel.IconAuto, map[string]string{"LANG": "en_US.UTF-8"}, false, datamodel.IconText},
		{"auto on a tty text on dumb terminal", datamodel.IconAuto, map[string]string{"TERM": "dumb"}, true, datamodel.IconText},
		{"auto on a tty text on non-utf8 locale", datamodel.IconAuto, map[string]string{"LANG": "C"}, true, datamodel.IconText},
		{"auto text when LC_ALL non-utf8 overrides LANG", datamodel.IconAuto, map[string]string{"LC_ALL": "C", "LANG": "en_US.UTF-8"}, true, datamodel.IconText},
		{"env garbage ignored then default nerd on a tty", datamodel.IconAuto, map[string]string{"KIRA_ICONS": "bogus", "LC_ALL": "C.utf8"}, true, datamodel.IconNerd},
		{"unknown config treated as auto", datamodel.IconMode("weird"), map[string]string{"LANG": "C"}, true, datamodel.IconText},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolveIconMode(c.mode, envFrom(c.env), c.tty); got != c.want {
				t.Errorf("resolveIconMode(%q, %v, tty=%v) = %q, want %q", c.mode, c.env, c.tty, got, c.want)
			}
		})
	}
}

func TestUTF8Locale(t *testing.T) {
	t.Parallel()
	cases := []struct {
		env  map[string]string
		want bool
	}{
		{map[string]string{"LC_ALL": "en_US.UTF-8"}, true},
		{map[string]string{"LC_CTYPE": "de_DE.utf8"}, true},
		{map[string]string{"LANG": "fr_FR.UTF-8"}, true},
		{map[string]string{"LANG": "C"}, false},
		{map[string]string{"LC_ALL": "C", "LANG": "en_US.UTF-8"}, false},
		{nil, false},
	}
	for _, c := range cases {
		if got := utf8Locale(envFrom(c.env)); got != c.want {
			t.Errorf("utf8Locale(%v) = %v, want %v", c.env, got, c.want)
		}
	}
}
