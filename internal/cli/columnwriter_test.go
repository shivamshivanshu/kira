package cli

import (
	"bytes"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/charmbracelet/x/ansi"
)

func TestColumnWriterMatchesTabwriterOnASCII(t *testing.T) {
	inputs := []string{
		"",
		"no tabs\n",
		"KEY\tSTATE\tTITLE\nKIRA-1\tTODO\tfirst\nKIRA-12\tIN_PROGRESS\tsecond\n",
		"header line\na\tb\tc\nlonger\tx\n\tempty first\nplain\nzz\tyy\tww\tvv\n",
		"trailing\ttab\t\nx\t\t\n",
		"no final newline\tcell\nab\tcd",
	}
	for _, in := range inputs {
		var want, got bytes.Buffer
		std := tabwriter.NewWriter(&want, 0, 4, columnPadding, ' ', 0)
		_, _ = std.Write([]byte(in))
		_ = std.Flush()
		cw := newTabWriter(&got)
		_, _ = cw.Write([]byte(in))
		_ = cw.Flush()
		if got.String() != want.String() {
			t.Errorf("input %q:\n got %q\nwant %q", in, got.String(), want.String())
		}
	}
}

func TestColumnWriterAlignsWideCells(t *testing.T) {
	var out bytes.Buffer
	cw := newTabWriter(&out)
	_, _ = cw.Write([]byte("KIRA-1\t标题标题\tTODO\nKIRA-2\tascii\tDONE\n"))
	_ = cw.Flush()
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	stateColumn := func(line string) int {
		return ansi.StringWidth(line[:strings.LastIndex(line, " ")+1])
	}
	if stateColumn(lines[0]) != stateColumn(lines[1]) {
		t.Errorf("state column misaligned:\n%s", out.String())
	}
}
