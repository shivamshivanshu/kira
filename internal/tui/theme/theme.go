// Package theme is kira's single source of terminal color: every color literal in the codebase lives here.
package theme

import (
	"io"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

var (
	accent        = lipgloss.AdaptiveColor{Light: "#268bd2", Dark: "#7dcfff"}
	dim           = lipgloss.AdaptiveColor{Light: "#8a8a8a", Dark: "#6c7086"}
	border        = lipgloss.AdaptiveColor{Light: "#b0b0b0", Dark: "#3b4261"}
	categoryTodo  = lipgloss.AdaptiveColor{Light: "#6c6c6c", Dark: "#a9b1d6"}
	categoryDoing = lipgloss.AdaptiveColor{Light: "#b58900", Dark: "#e0af68"}
	categoryDone  = lipgloss.AdaptiveColor{Light: "#28a745", Dark: "#9ece6a"}
	priorityP0    = lipgloss.AdaptiveColor{Light: "#d70000", Dark: "#f7768e"}
	heatWarm      = categoryDoing
	heatHot       = priorityP0
)

type Heat struct {
	Warm lipgloss.Style
	Hot  lipgloss.Style
}

type Theme struct {
	r *lipgloss.Renderer

	Text   lipgloss.Style
	Accent lipgloss.Style
	Dim    lipgloss.Style
	Border lipgloss.Style
	Heat   Heat

	category map[datamodel.Category]lipgloss.Style
}

func New(r *lipgloss.Renderer) Theme { return newTheme(r, nil) }

func newTheme(r *lipgloss.Renderer, overrides map[string]string) Theme {
	p := palette(overrides)
	fg := func(c lipgloss.TerminalColor) lipgloss.Style { return r.NewStyle().Foreground(c) }
	return Theme{
		r:      r,
		Text:   r.NewStyle(),
		Accent: fg(p.accent),
		Dim:    fg(p.dim),
		Border: fg(p.border),
		category: map[datamodel.Category]lipgloss.Style{
			datamodel.CategoryTodo:  fg(p.todo),
			datamodel.CategoryDoing: fg(p.doing),
			datamodel.CategoryDone:  fg(p.done),
		},
		Heat: Heat{Warm: fg(p.warm), Hot: fg(p.hot)},
	}
}

type themePalette struct {
	accent, dim, border, todo, doing, done, warm, hot lipgloss.AdaptiveColor
}

func palette(overrides map[string]string) themePalette {
	p := themePalette{accent: accent, dim: dim, border: border, todo: categoryTodo, doing: categoryDoing, done: categoryDone, warm: heatWarm, hot: heatHot}
	slots := paletteSlots(&p)
	for slot, hex := range overrides {
		if dst, ok := slots[slot]; ok && datamodel.IsHexColor(hex) {
			*dst = lipgloss.AdaptiveColor{Light: hex, Dark: hex}
		}
	}
	return p
}

func paletteSlots(p *themePalette) map[string]*lipgloss.AdaptiveColor {
	return map[string]*lipgloss.AdaptiveColor{
		"accent": &p.accent, "dim": &p.dim, "border": &p.border, "todo": &p.todo,
		"doing": &p.doing, "done": &p.done, "warm": &p.warm, "hot": &p.hot,
	}
}

func (t Theme) CategoryStyle(c datamodel.Category) lipgloss.Style {
	if s, ok := t.category[c]; ok {
		return s
	}
	return t.Text
}

func (t Theme) Renderer() *lipgloss.Renderer { return t.r }

func For(w io.Writer, ui datamodel.UI, noColor bool) Theme {
	return newTheme(NewRenderer(w, ui, noColor), ui.Theme)
}

func NewRenderer(w io.Writer, ui datamodel.UI, noColor bool) *lipgloss.Renderer {
	r := lipgloss.NewRenderer(w)
	switch resolveColor(ui.Color, noColor) {
	case colorNever:
		r.SetColorProfile(termenv.Ascii)
	case colorAlways:
		r.SetColorProfile(termenv.TrueColor)
	}
	r.SetHasDarkBackground(prefersDark(ui.Background))
	return r
}

type colorDecision int

const (
	colorAuto colorDecision = iota
	colorNever
	colorAlways
)

func resolveColor(mode datamodel.ColorMode, noColor bool) colorDecision {
	if noColor || os.Getenv("NO_COLOR") != "" {
		return colorNever
	}
	switch mode {
	case datamodel.ColorAlways:
		return colorAlways
	case datamodel.ColorNever:
		return colorNever
	default:
		return colorAuto
	}
}

func prefersDark(bg datamodel.Background) bool {
	return bg != datamodel.BackgroundLight
}
