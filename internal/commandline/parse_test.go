package commandline

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		line string
		want []string
	}{
		{"", nil},
		{" \t\n\r\u2003", nil},
		{"list\t--owner\n名\u2003--state\rOPEN", []string{"list", "--owner", "名", "--state", "OPEN"}},
		{`edit . --title "order book merge" 'p3'`, []string{"edit", ".", "--title", "order book merge", "p3"}},
		{`comment . -m "say \"hi\""`, []string{"comment", ".", "-m", `say "hi"`}},
		{`edit . --title hello\ world`, []string{"edit", ".", "--title", "hello world"}},
		{`edit . --title hello\ `, []string{"edit", ".", "--title", "hello "}},
		{`comment . -m 'C:\path\'`, []string{"comment", ".", "-m", `C:\path\`}},
		{`edit . --owner "" --title pre'fix'"suffix"`, []string{"edit", ".", "--owner", "", "--title", "prefixsuffix"}},
		{`'' ""`, []string{"", ""}},
		{`comment . -m '$(touch /tmp/no)'`, []string{"comment", ".", "-m", "$(touch /tmp/no)"}},
		{"show $USER ~ `date` | cat > output", []string{"show", "$USER", "~", "`date`", "|", "cat", ">", "output"}},
	} {
		t.Run(tc.line, func(t *testing.T) {
			got, err := Parse(tc.line)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Parse(%q) = %#v, %v; want %#v, nil", tc.line, got, err, tc.want)
			}
		})
	}
}

func TestParseRejectsUnfinishedInput(t *testing.T) {
	t.Parallel()
	for _, line := range []string{`comment . -m "unfinished`, `comment . -m 'unfinished`, `show .\`, `comment . -m "unfinished\`} {
		t.Run(line, func(t *testing.T) {
			got, err := Parse(line)
			if err == nil || got != nil {
				t.Fatalf("Parse(%q) = %#v, %v; want no arguments and an error", line, got, err)
			}
		})
	}
}
