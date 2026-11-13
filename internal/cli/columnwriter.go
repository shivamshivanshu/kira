package cli

import (
	"bytes"
	"io"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const columnPadding = 2

// columnWriter reproduces text/tabwriter's column-block layout (minwidth 0,
// padding 2, space padding, no flags) but measures cells in terminal display
// cells. tabwriter counts runes, so a wide (e.g. CJK) cell shifts every column
// to its right.
type columnWriter struct {
	out io.Writer
	buf bytes.Buffer
}

func newTabWriter(w io.Writer) *columnWriter { return &columnWriter{out: w} }

func (c *columnWriter) Write(p []byte) (int, error) { return c.buf.Write(p) }

func (c *columnWriter) Flush() error {
	text := c.buf.String()
	c.buf.Reset()
	if text == "" {
		return nil
	}
	endsWithNewline := strings.HasSuffix(text, "\n")
	rawLines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	lines := make([][]string, len(rawLines))
	for i, l := range rawLines {
		lines[i] = strings.Split(l, "\t")
	}
	var out strings.Builder
	layoutBlocks(&out, lines, nil, 0, len(lines))
	rendered := out.String()
	if !endsWithNewline {
		rendered = strings.TrimSuffix(rendered, "\n")
	}
	_, err := io.WriteString(c.out, rendered)
	return err
}

// layoutBlocks mirrors tabwriter's recursive format: a column block is a run of
// consecutive lines that all have a tab-terminated cell in the current column;
// its width is the widest such cell plus padding, and nested columns are laid
// out within the block.
func layoutBlocks(out *strings.Builder, lines [][]string, widths []int, line0, line1 int) {
	column := len(widths)
	for this := line0; this < line1; this++ {
		if column >= len(lines[this])-1 {
			continue
		}
		writeLines(out, lines[line0:this], widths)
		line0 = this
		width := 0
		for ; this < line1 && column < len(lines[this])-1; this++ {
			width = max(width, ansi.StringWidth(lines[this][column])+columnPadding)
		}
		layoutBlocks(out, lines, append(widths, width), line0, this)
		line0 = this
	}
	writeLines(out, lines[line0:line1], widths)
}

func writeLines(out *strings.Builder, lines [][]string, widths []int) {
	for _, cells := range lines {
		for j, cell := range cells {
			out.WriteString(cell)
			if j < len(widths) {
				out.WriteString(strings.Repeat(" ", widths[j]-ansi.StringWidth(cell)))
			}
		}
		out.WriteByte('\n')
	}
}
