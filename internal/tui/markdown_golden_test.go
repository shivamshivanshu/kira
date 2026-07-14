package tui

import (
	"testing"

	"github.com/charmbracelet/x/exp/golden"
)

type mdCase struct {
	name  string
	width int
	body  string
}

var mdCorpus = []mdCase{
	{"heading-atx", 40, "# Heading One\n\n## Heading Two\n\n### H3\n#### H4\n##### H5\n###### H6"},
	{"heading-setext", 40, "Title Here\n==========\n\nSub Title\n---------"},
	{"paragraph", 40, "Just a simple paragraph of text that stays on one line."},
	{"emphasis", 40, "Some *italic* and **bold** and ***both*** and `code` words."},
	{"strikethrough", 40, "This is ~~struck~~ text."},
	{"list-unordered", 40, "- one\n- two\n- three"},
	{"list-unordered-nested", 40, "- top\n  - child a\n  - child b\n- top2"},
	{"list-ordered", 40, "1. first\n2. second\n3. third"},
	{"list-ordered-nested", 40, "1. first\n   1. sub one\n   2. sub two\n2. second"},
	{"list-task", 100, "- [ ] TSan clean on order_book_test\n- [x] done item\n- [ ] No p99 regression"},
	{"code-fenced", 40, "```go\nfunc main() {\n\tprintln(\"hi\")\n}\n```"},
	{"code-fenced-nolang", 40, "```\nplain code\n  indented\n```"},
	{"code-indented", 40, "    indented code block\n    line two"},
	{"blockquote", 40, "> a block quote\n> second line"},
	{"blockquote-nested", 40, "> outer\n>> inner"},
	{"link", 40, "See [the docs](https://example.com/docs) for more."},
	{"link-autolink", 40, "Visit https://example.com now."},
	{"link-long-url", 40, "See https://example.com/very/long/path/segment/that/exceeds/the/wrap/width for details."},
	{"image", 40, "![architecture diagram](https://example.com/diagram.png)"},
	{"thematic-break", 40, "above\n\n---\n\nbelow"},
	{"hard-break", 40, "line one  \nline two"},
	{"wrap-narrow", 40, "The quick brown fox jumps over the lazy dog and then keeps running across the meadow toward the distant hills."},
	{"wrap-wide", 80, "The quick brown fox jumps over the lazy dog and then keeps running across the meadow toward the distant hills."},
	{"mixed", 60, "# Title\n\nIntro paragraph here.\n\n## Section\n\n- item **bold**\n- item with `code`\n\n> note\n\n```\nblock\n```\n\nDone."},
	{"ticket-body", 100, "## Description\n\nThe snapshot merge path drops updates when two feed threads race on the\nsame price level. Repro: `bench/burst_test --dup-updates=high`.\n\n## Acceptance criteria\n- [ ] TSan clean on order_book_test\n- [ ] No p99 regression on hot path"},
}

func TestMarkdownGolden(t *testing.T) {
	t.Parallel()
	for _, c := range mdCorpus {
		t.Run(c.name, func(t *testing.T) {
			golden.RequireEqual(t, []byte(renderMarkdown(c.body, c.width)))
		})
	}
}
