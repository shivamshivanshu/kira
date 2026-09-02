package core

import (
	"maps"
	"os"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/editorx"
	"github.com/shivamshivanshu/kira/internal/entityschema"
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
	Parent   *string  `yaml:"parent"`
	Sprint   *string  `yaml:"sprint"`
	Due      *string  `yaml:"due"`
	Estimate *float64 `yaml:"estimate"`

	ExtraScalars map[string]datamodel.Rendered   `yaml:"-"`
	ExtraLists   map[string][]datamodel.Rendered `yaml:"-"`
	ExtraBlocks  map[string]datamodel.Rendered   `yaml:"-"`
	Body         string                          `yaml:"-"`
}

// parseDraft captures the non-builtin frontmatter alongside the builtin fields
// so a template or --from-file document carries its extras into the item
// instead of losing them on the way through.
func parseDraft(content string) (draft, error) {
	var d draft
	body, err := codec.DecodeFrontmatter(content, &d)
	if err != nil {
		return draft{}, err
	}
	scalars, lists, blocks, err := codec.ExtraFrontmatter(content)
	if err != nil {
		return draft{}, err
	}
	d.ExtraScalars, d.ExtraLists, d.ExtraBlocks, d.Body = scalars, lists, blocks, body
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
	b.WriteString("parent: " + draftScalar(ptr.Deref(d.Parent)) + "\n")
	b.WriteString("sprint: " + draftScalar(ptr.Deref(d.Sprint)) + "\n")
	b.WriteString("due: " + draftScalar(ptr.Deref(d.Due)) + "\n")
	estimate := ""
	if d.Estimate != nil {
		estimate = codec.EmitFloat(*d.Estimate)
	}
	b.WriteString("estimate: " + estimate + "\n")
	writeDraftExtras(&b, d)
	b.WriteString(codec.FenceLine)
	b.WriteString(d.Body)
	return b.String()
}

// writeDraftExtras renders an offered-but-unset field as a blank key rather
// than the literal null an empty YAML scalar parses back to. A draft is a
// buffer to fill in, so "severity:" invites a value where "severity: null"
// looks like one. Items are written by codec.Serialize, which is unaffected.
func writeDraftExtras(b *strings.Builder, d draft) {
	scalars := maps.Clone(d.ExtraScalars)
	for name, v := range scalars {
		if v == "null" {
			scalars[name] = ""
		}
	}
	codec.WriteExtras(b, scalars, d.ExtraLists, d.ExtraBlocks)
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
	if _, after, ok := strings.Cut(content, bannerClose); ok {
		return strings.TrimLeft(after, "\n")
	}
	return content
}

func runEditor(editor string, stdio editorx.Stdio, initial string, validate func(content string) []error) (string, error) {
	ed, err := editorx.Command(editor)
	if err != nil {
		return "", errx.Env("%w", err).WithHint(editorx.ConfigHint)
	}
	tmp, err := os.CreateTemp("", "kira-*.md")
	if err != nil {
		return "", errx.Env("creating editor buffer: %v", err)
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(path) }()

	buffer := initial
	annotated := false
	for {
		if err := os.WriteFile(path, []byte(buffer), 0o600); err != nil {
			return "", errx.Env("writing editor buffer: %v", err)
		}
		if err := ed.Edit(path, stdio); err != nil {
			return "", errx.User("%w", err)
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

func defaultTemplate(typ string, schema entityschema.Schema) string {
	var body strings.Builder
	d := draft{Type: typ}
	for _, f := range schema.Fields {
		switch {
		case f.Placement == entityschema.PlacementBody && f.Section != "":
			body.WriteString("\n## " + f.Section + "\n")
		case f.IsCustomFrontmatter():
			offerExtra(&d, f)
		}
	}
	d.Body = body.String()
	return serializeDraft(d)
}

// offerExtra puts a declared field in the draft empty, so the editor buffer and
// --print-template show every key the type accepts. Left empty it reads back as
// unset, the same as an untouched builtin key.
func offerExtra(d *draft, f entityschema.FieldDef) {
	if f.List {
		if d.ExtraLists == nil {
			d.ExtraLists = map[string][]datamodel.Rendered{}
		}
		d.ExtraLists[f.Name] = nil
		return
	}
	if d.ExtraScalars == nil {
		d.ExtraScalars = map[string]datamodel.Rendered{}
	}
	d.ExtraScalars[f.Name] = ""
}

// dropUnsetDeclaredExtras removes a declared field the draft carries with no
// value. An offered key left untouched must mean unset, not the literal null
// that an empty YAML scalar parses to. Undeclared keys are never dropped --
// they round-trip verbatim so a newer kira's fields survive an older one.
func dropUnsetDeclaredExtras(d draft, schema entityschema.Schema) draft {
	// Taking d by value copies the map headers, not the maps, so deleting
	// in place would reach back into the caller's draft. Clone to keep the
	// value-in/value-out shape honest.
	d.ExtraScalars = maps.Clone(d.ExtraScalars)
	d.ExtraLists = maps.Clone(d.ExtraLists)
	for name, v := range d.ExtraScalars {
		if v != "" && v != "null" {
			continue
		}
		if f, ok := schema.Field(name); ok && f.IsCustomFrontmatter() {
			delete(d.ExtraScalars, name)
		}
	}
	for name, v := range d.ExtraLists {
		if len(v) > 0 {
			continue
		}
		if f, ok := schema.Field(name); ok && f.IsCustomFrontmatter() {
			delete(d.ExtraLists, name)
		}
	}
	return d
}
