package core

import (
	"os"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/editorx"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

type draft struct {
	Title    string   `yaml:"title"`
	Type     string   `yaml:"type"`
	Subtype  *string  `yaml:"subtype"`
	Priority *string  `yaml:"priority"`
	Rank     *string  `yaml:"rank"`
	Owner    *string  `yaml:"owner"`
	Reporter *string  `yaml:"reporter"`
	Labels   []string `yaml:"labels"`
	Epic     *string  `yaml:"epic"`
	Sprint   *string  `yaml:"sprint"`
	Due      *string  `yaml:"due"`
	Estimate *float64 `yaml:"estimate"`

	Body string `yaml:"-"`
}

func parseDraft(content string) (draft, error) {
	var d draft
	body, err := codec.DecodeFrontmatter(content, &d)
	if err != nil {
		return draft{}, err
	}
	d.Body = body
	return d, nil
}

func serializeDraft(d draft) string {
	var b strings.Builder
	b.WriteString(codec.FenceLine)
	b.WriteString("title: " + draftScalar(d.Title) + "\n")
	b.WriteString("type: " + draftScalar(d.Type) + "\n")
	b.WriteString("subtype: " + draftScalar(ptr.Deref(d.Subtype)) + "\n")
	b.WriteString("priority: " + draftScalar(ptr.Deref(d.Priority)) + "\n")
	b.WriteString("rank: " + draftScalar(ptr.Deref(d.Rank)) + "\n")
	b.WriteString("owner: " + draftScalar(ptr.Deref(d.Owner)) + "\n")
	b.WriteString("reporter: " + draftScalar(ptr.Deref(d.Reporter)) + "\n")
	b.WriteString("labels: " + codec.EmitList(d.Labels) + "\n")
	b.WriteString("epic: " + draftScalar(ptr.Deref(d.Epic)) + "\n")
	b.WriteString("sprint: " + draftScalar(ptr.Deref(d.Sprint)) + "\n")
	b.WriteString("due: " + draftScalar(ptr.Deref(d.Due)) + "\n")
	estimate := ""
	if d.Estimate != nil {
		estimate = codec.EmitFloat(*d.Estimate)
	}
	b.WriteString("estimate: " + estimate + "\n")
	b.WriteString(codec.FenceLine)
	b.WriteString(d.Body)
	return b.String()
}

func draftScalar(s string) string {
	if s == "" {
		return ""
	}
	return codec.EmitScalar(s)
}

const (
	bannerOpen  = "<!-- kira:error"
	bannerClose = "-->\n"
)

func errorBanner(errs []error) string {
	var b strings.Builder
	b.WriteString(bannerOpen + "\n")
	for _, e := range errs {
		b.WriteString(e.Error() + "\n")
	}
	b.WriteString(bannerClose + "\n")
	return b.String()
}

func stripErrorBanner(content string) string {
	if !strings.HasPrefix(content, bannerOpen) {
		return content
	}
	if i := strings.Index(content, bannerClose); i >= 0 {
		return strings.TrimLeft(content[i+len(bannerClose):], "\n")
	}
	return content
}

func runEditor(editor string, stdio editorx.Stdio, initial string, validate func(content string) []error) (string, error) {
	if _, err := editorx.Command(editor); err != nil {
		return "", errx.Env("%v", err).WithHint("set ui.editor in config or `export EDITOR=vim`")
	}
	tmp, err := os.CreateTemp("", "kira-*.md")
	if err != nil {
		return "", errx.Env("creating editor buffer: %v", err)
	}
	path := tmp.Name()
	tmp.Close()
	defer os.Remove(path)

	buffer := initial
	annotated := false
	for {
		if err := os.WriteFile(path, []byte(buffer), 0o600); err != nil {
			return "", errx.Env("writing editor buffer: %v", err)
		}
		if err := editorx.Edit(editor, path, stdio); err != nil {
			return "", errx.User("%v", err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", errx.User("reading editor buffer: %v", err)
		}
		edited := string(raw)
		content := stripErrorBanner(edited)
		errs := validate(content)
		if len(errs) == 0 {
			return content, nil
		}
		if annotated && edited == buffer {
			return "", errx.User("aborted: %d validation error(s), buffer unchanged", len(errs))
		}
		buffer = errorBanner(errs) + content
		annotated = true
	}
}

func defaultTemplate(typ string) string {
	return serializeDraft(draft{
		Type: typ,
		Body: "\n## Description\n\n## Acceptance criteria\n\n## Comments\n",
	})
}
