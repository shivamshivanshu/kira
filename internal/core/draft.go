package core

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/item"
)

// draft is the user-editable subset of an item, as it appears in a $EDITOR
// buffer or a --from-file document for create: the system fields (id, number,
// state, created, updated, blocked_by) are assigned by core, never edited, so
// they are absent from the draft form (docs/design/04-cli.md §6, and the
// --print-template shape). It is deliberately distinct from item.Item, whose
// Parse requires a fully-formed file.
type draft struct {
	Title    string   `yaml:"title"`
	Type     string   `yaml:"type"`
	Priority *string  `yaml:"priority"`
	Owner    *string  `yaml:"owner"`
	Reporter *string  `yaml:"reporter"`
	Labels   []string `yaml:"labels"`
	Epic     *string  `yaml:"epic"`
	Estimate *float64 `yaml:"estimate"`

	Body string `yaml:"-"`
}

func parseDraft(content string) (draft, error) {
	front, body, err := item.SplitDocument(content)
	if err != nil {
		return draft{}, err
	}
	var d draft
	if err := yaml.Unmarshal([]byte(front), &d); err != nil {
		return draft{}, fmt.Errorf("frontmatter yaml: %w", err)
	}
	d.Body = body
	return d, nil
}

// serializeDraft renders a draft in the fixed, one-key-per-line skeleton that
// the editor buffer and templates use. Optional scalars render as an empty
// value (`owner: `) rather than being omitted, so the skeleton shows every
// editable field; an empty value re-parses as unset.
func serializeDraft(d draft) string {
	var b strings.Builder
	b.WriteString(item.FenceLine)
	b.WriteString("title: " + draftScalar(d.Title) + "\n")
	b.WriteString("type: " + draftScalar(d.Type) + "\n")
	b.WriteString("priority: " + draftScalarPtr(d.Priority) + "\n")
	b.WriteString("owner: " + draftScalarPtr(d.Owner) + "\n")
	b.WriteString("reporter: " + draftScalarPtr(d.Reporter) + "\n")
	b.WriteString("labels: " + item.EmitList(d.Labels) + "\n")
	b.WriteString("epic: " + draftScalarPtr(d.Epic) + "\n")
	estimate := ""
	if d.Estimate != nil {
		estimate = item.EmitFloat(*d.Estimate)
	}
	b.WriteString("estimate: " + estimate + "\n")
	b.WriteString(item.FenceLine)
	b.WriteString(d.Body)
	return b.String()
}

// draftScalar renders a scalar via the item codec's canonical emitter, except
// an empty value renders blank (an unset editable field), not `""`.
func draftScalar(s string) string {
	if s == "" {
		return ""
	}
	return item.EmitScalar(s)
}

func draftScalarPtr(p *string) string {
	if p == nil {
		return ""
	}
	return draftScalar(*p)
}

const (
	bannerOpen  = "<!-- kira:error"
	bannerClose = "-->\n"
)

// errorBanner formats validation errors as an HTML-comment block prepended to
// the editor buffer above the frontmatter (docs/design/04-cli.md §6). It is
// stripped before the next parse by stripErrorBanner.
func errorBanner(errs []error) string {
	var b strings.Builder
	b.WriteString(bannerOpen + "\n")
	for _, e := range errs {
		b.WriteString(e.Error() + "\n")
	}
	b.WriteString(bannerClose + "\n")
	return b.String()
}

// stripErrorBanner removes a leading error banner (if any) so the remaining
// document parses as an ordinary draft/item.
func stripErrorBanner(content string) string {
	if !strings.HasPrefix(content, bannerOpen) {
		return content
	}
	if i := strings.Index(content, bannerClose); i >= 0 {
		return strings.TrimLeft(content[i+len(bannerClose):], "\n")
	}
	return content
}

// defaultTemplate is the built-in skeleton init writes to
// templates/<type>.md; users may customize it thereafter. It is a draft with
// only the type fixed, plus the three canonical body sections
// (docs/design/02-data-model.md §1).
func defaultTemplate(typ string) string {
	return serializeDraft(draft{
		Type: typ,
		Body: "\n## Description\n\n## Acceptance criteria\n\n## Comments\n",
	})
}
