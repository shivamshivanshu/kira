package theme_test

import (
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

func pinnedRenderer(dark bool) *lipgloss.Renderer {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	r.SetHasDarkBackground(dark)
	return r
}

func TestNoColorRendersPlain(t *testing.T) {
	th := theme.New(theme.NewRenderer(io.Discard, datamodel.UI{Background: datamodel.BackgroundDark}, true))
	got := th.CategoryStyle(datamodel.CategoryDone).Render("DONE")
	if got != "DONE" {
		t.Errorf("no-color render = %q, want plain %q", got, "DONE")
	}
}

func TestAdaptiveColorFollowsBackground(t *testing.T) {
	dark := theme.New(pinnedRenderer(true)).CategoryStyle(datamodel.CategoryDone).Render("x")
	light := theme.New(pinnedRenderer(false)).CategoryStyle(datamodel.CategoryDone).Render("x")
	if dark == light {
		t.Fatalf("dark and light renders identical (%q); AdaptiveColor not resolving per background", dark)
	}
	for _, s := range []string{dark, light} {
		if !strings.Contains(s, "\x1b[") {
			t.Errorf("expected ANSI color in %q under TrueColor profile", s)
		}
	}
}
