package tui

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestGlyphPickExhaustive(t *testing.T) {
	t.Parallel()
	g := glyph{nerd: "N", emoji: "E", ascii: "A"}
	want := map[datamodel.IconMode]string{
		datamodel.IconNerd:   "N",
		datamodel.IconEmoji:  "E",
		datamodel.IconText:   "A",
		datamodel.IconAuto:   "A",
		datamodel.IconAlways: "A",
		datamodel.IconNever:  "A",
	}
	for _, m := range datamodel.IconModes {
		exp, ok := want[m]
		if !ok {
			t.Fatalf("no expected glyph for %q; add a case", m)
		}
		if got := g.pick(m); got != exp {
			t.Errorf("pick(%q) = %q, want %q", m, got, exp)
		}
	}
}

func TestCanonicalIconModeExhaustive(t *testing.T) {
	t.Parallel()
	type res struct {
		mode datamodel.IconMode
		ok   bool
	}
	want := map[datamodel.IconMode]res{
		datamodel.IconNerd:   {datamodel.IconNerd, true},
		datamodel.IconAlways: {datamodel.IconNerd, true},
		datamodel.IconEmoji:  {datamodel.IconEmoji, true},
		datamodel.IconText:   {datamodel.IconText, true},
		datamodel.IconNever:  {datamodel.IconText, true},
		datamodel.IconAuto:   {"", false},
	}
	for _, m := range datamodel.IconModes {
		exp, ok := want[m]
		if !ok {
			t.Fatalf("no expected canonicalization for %q; add a case", m)
		}
		got, gotOK := canonicalIconMode(m)
		if got != exp.mode || gotOK != exp.ok {
			t.Errorf("canonicalIconMode(%q) = (%q,%v), want (%q,%v)", m, got, gotOK, exp.mode, exp.ok)
		}
	}
}

func TestCategoryGlyphExhaustive(t *testing.T) {
	t.Parallel()
	ic := iconSet{mode: datamodel.IconEmoji}
	want := map[datamodel.Category]string{
		datamodel.CategoryTodo:  glyphTodo.emoji,
		datamodel.CategoryDoing: glyphDoing.emoji,
		datamodel.CategoryDone:  glyphDone.emoji,
	}
	for _, c := range datamodel.Categories {
		exp, ok := want[c]
		if !ok {
			t.Fatalf("no expected glyph for %q; add a case", c)
		}
		if got := ic.categoryGlyph(c, nil); got != exp {
			t.Errorf("categoryGlyph(%q) = %q, want %q", c, got, exp)
		}
	}
}
