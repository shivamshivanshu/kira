package tui

import (
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

const mdMargin = 2

var (
	mdMu     sync.Mutex
	mdCache  = map[int]*mdRenderer{}
	mdParser = goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser()
)

func renderMarkdown(body string, width int) string {
	body = strings.TrimRight(body, "\n")
	if width <= 0 || body == "" {
		return body
	}
	return markdownRenderer(width).render(body)
}

func markdownRenderer(width int) *mdRenderer {
	mdMu.Lock()
	defer mdMu.Unlock()
	if r, ok := mdCache[width]; ok {
		return r
	}
	r := &mdRenderer{width: width}
	mdCache[width] = r
	return r
}

type mdRenderer struct {
	width int
}

func (r *mdRenderer) render(body string) string {
	src := []byte(body)
	doc := mdParser.Parse(text.NewReader(src))
	w := &mdWalk{src: src, width: r.width}
	lines := w.blocks(doc, mdMargin)
	if hasTopMargin(doc.FirstChild()) {
		lines = append([]string{""}, lines...)
	}
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.Join(lines, "\n")
}

func hasTopMargin(n ast.Node) bool {
	switch n.(type) {
	case *ast.List, *ast.Blockquote, *ast.FencedCodeBlock, *ast.CodeBlock:
		return true
	}
	return false
}

type mdWalk struct {
	src   []byte
	width int
}

func (w *mdWalk) blocks(parent ast.Node, indent int) []string {
	var out []string
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		lines := w.block(c, indent)
		if len(lines) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, lines...)
	}
	return out
}

func (w *mdWalk) block(n ast.Node, indent int) []string {
	switch b := n.(type) {
	case *ast.Heading:
		return w.wrapPad(strings.Repeat("#", b.Level)+" "+w.inline(b), indent)
	case *ast.ThematicBreak:
		return []string{sp(indent) + "--------"}
	case *ast.FencedCodeBlock:
		return w.code(b.Lines(), indent)
	case *ast.CodeBlock:
		return w.code(b.Lines(), indent)
	case *ast.Blockquote:
		return w.quote(b, indent)
	case *ast.List:
		return w.list(b, indent)
	case *east.Table:
		return w.table(b, indent)
	default:
		return w.wrapPad(w.inline(n), indent)
	}
}

func (w *mdWalk) code(lines *text.Segments, indent int) []string {
	out := make([]string, 0, lines.Len())
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		out = append(out, sp(indent+2)+strings.TrimRight(string(seg.Value(w.src)), "\n"))
	}
	return out
}

func (w *mdWalk) quote(n ast.Node, indent int) []string {
	inner := w.blocks(n, indent+2)
	out := make([]string, 0, len(inner))
	for _, l := range inner {
		if strings.TrimSpace(l) == "" {
			out = append(out, sp(indent)+"|")
			continue
		}
		out = append(out, sp(indent)+"| "+l[indent+2:])
	}
	return out
}

func (w *mdWalk) list(n *ast.List, indent int) []string {
	num := n.Start
	if num == 0 {
		num = 1
	}
	var out []string
	for item := n.FirstChild(); item != nil; item = item.NextSibling() {
		marker := "• "
		if n.IsOrdered() {
			marker = strconv.Itoa(num) + ". "
			num++
		}
		out = append(out, w.item(item, indent, marker)...)
	}
	return out
}

func (w *mdWalk) item(li ast.Node, indent int, marker string) []string {
	if isTaskItem(li) {
		marker = ""
	}
	mw := ansi.StringWidth(marker)
	var out []string
	child := li.FirstChild()
	if isTextish(child) {
		segs := w.wrap(w.inline(child), w.width-indent-mw)
		for i, s := range segs {
			if i == 0 {
				out = append(out, sp(indent)+marker+s)
			} else {
				out = append(out, sp(indent+mw)+s)
			}
		}
		child = child.NextSibling()
	}
	for ; child != nil; child = child.NextSibling() {
		out = append(out, w.block(child, indent+4)...)
	}
	return out
}

func (w *mdWalk) table(n ast.Node, indent int) []string {
	var out []string
	first := true
	for row := n.FirstChild(); row != nil; row = row.NextSibling() {
		var cells []string
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			cells = append(cells, strings.TrimSpace(w.inline(cell)))
		}
		out = append(out, sp(indent)+strings.Join(cells, " | "))
		if first {
			seps := make([]string, len(cells))
			for i, c := range cells {
				seps[i] = strings.Repeat("-", max(1, ansi.StringWidth(c)))
			}
			out = append(out, sp(indent)+strings.Join(seps, "-+-"))
			first = false
		}
	}
	return out
}

func (w *mdWalk) inline(n ast.Node) string {
	var b strings.Builder
	w.inlineChildren(&b, n)
	return b.String()
}

func (w *mdWalk) inlineChildren(b *strings.Builder, n ast.Node) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		w.inlineNode(b, c)
	}
}

func (w *mdWalk) inlineNode(b *strings.Builder, n ast.Node) {
	switch t := n.(type) {
	case *ast.Text:
		b.Write(t.Segment.Value(w.src))
		if t.SoftLineBreak() || t.HardLineBreak() {
			b.WriteByte(' ')
		}
	case *ast.String:
		b.Write(t.Value)
	case *ast.CodeSpan:
		w.inlineChildren(b, t)
	case *ast.Emphasis:
		m := strings.Repeat("*", t.Level)
		b.WriteString(m)
		w.inlineChildren(b, t)
		b.WriteString(m)
	case *ast.Link:
		w.inlineChildren(b, t)
		if len(t.Destination) > 0 {
			b.WriteByte(' ')
			b.Write(t.Destination)
		}
	case *ast.AutoLink:
		b.Write(t.URL(w.src))
	case *ast.Image:
		w.inlineChildren(b, t)
		if len(t.Destination) > 0 {
			b.WriteByte(' ')
			b.Write(t.Destination)
		}
	case *east.Strikethrough:
		b.WriteString("~~")
		w.inlineChildren(b, t)
		b.WriteString("~~")
	case *east.TaskCheckBox:
		if t.IsChecked {
			b.WriteString("[x] ")
		} else {
			b.WriteString("[ ] ")
		}
	case *ast.RawHTML:
	default:
		w.inlineChildren(b, n)
	}
}

func (w *mdWalk) wrapPad(s string, indent int) []string {
	segs := w.wrap(s, w.width-indent)
	for i := range segs {
		segs[i] = sp(indent) + segs[i]
	}
	return segs
}

func (w *mdWalk) wrap(s string, budget int) []string {
	if budget < 1 {
		budget = 1
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	lines := []string{words[0]}
	for _, word := range words[1:] {
		last := len(lines) - 1
		if ansi.StringWidth(lines[last])+1+ansi.StringWidth(word) <= budget {
			lines[last] += " " + word
		} else {
			lines = append(lines, word)
		}
	}
	return lines
}

func isTaskItem(li ast.Node) bool {
	if fc := li.FirstChild(); fc != nil {
		_, ok := fc.FirstChild().(*east.TaskCheckBox)
		return ok
	}
	return false
}

func isTextish(n ast.Node) bool {
	switch n.(type) {
	case *ast.Paragraph, *ast.TextBlock:
		return true
	}
	return false
}

func sp(n int) string { return strings.Repeat(" ", n) }
