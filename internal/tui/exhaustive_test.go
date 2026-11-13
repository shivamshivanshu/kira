package tui

import (
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func TestGlyphPickExhaustive(t *testing.T) {
	t.Parallel()
	g := glyph{nerd: "N", emoji: "E", ascii: "A"}
	want := map[datamodel.IconMode]string{
		datamodel.IconNerd:  "N",
		datamodel.IconEmoji: "E",
		datamodel.IconText:  "A",
		datamodel.IconAuto:  "A",
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
