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

func hasANSI(s string) bool { return strings.Contains(s, "\x1b[") }

func TestColorAlwaysForcesANSIWhenPiped(t *testing.T) {
	th := theme.For(io.Discard, datamodel.UI{Color: datamodel.ColorAlways}, false)
	if !hasANSI(th.Accent.Render("x")) {
		t.Error("ui.color=always did not force ANSI on a non-TTY writer")
	}
}

func TestColorNeverAndPrecedence(t *testing.T) {
	if hasANSI(theme.For(io.Discard, datamodel.UI{Color: datamodel.ColorNever}, false).Accent.Render("x")) {
		t.Error("ui.color=never emitted ANSI")
	}
	if hasANSI(theme.For(io.Discard, datamodel.UI{Color: datamodel.ColorAlways}, true).Accent.Render("x")) {
		t.Error("--no-color did not override ui.color=always")
	}
	t.Setenv("NO_COLOR", "1")
	if hasANSI(theme.For(io.Discard, datamodel.UI{Color: datamodel.ColorAlways}, false).Accent.Render("x")) {
		t.Error("NO_COLOR did not override ui.color=always")
	}
}

func TestThemeSlotOverride(t *testing.T) {
	ui := datamodel.UI{Color: datamodel.ColorAlways, Theme: map[string]string{"accent": "#ff0000"}}
	got := theme.For(io.Discard, ui, false).Accent.Render("x")
	if !strings.Contains(got, "38;2;255;0;0") {
		t.Errorf("accent override not applied under TrueColor: %q", got)
	}
}

func TestBorderSlotOverride(t *testing.T) {
	ui := datamodel.UI{Color: datamodel.ColorAlways, Theme: map[string]string{"border": "#00ff00"}}
	got := theme.For(io.Discard, ui, false).Border.Render("x")
	if !strings.Contains(got, "38;2;0;255;0") {
		t.Errorf("border override not applied under TrueColor: %q", got)
	}
}

func TestThemeSlotInvalidKeepsDefault(t *testing.T) {
	base := datamodel.UI{Color: datamodel.ColorAlways}
	def := theme.For(io.Discard, base, false).Accent.Render("x")
	bad := base
	bad.Theme = map[string]string{"accent": "not-a-hex"}
	if got := theme.For(io.Discard, bad, false).Accent.Render("x"); got != def {
		t.Errorf("invalid hex should keep default: default=%q got=%q", def, got)
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
