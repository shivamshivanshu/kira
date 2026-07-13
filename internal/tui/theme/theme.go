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
	priorityP1    = lipgloss.AdaptiveColor{Light: "#d75f00", Dark: "#ff9e64"}
	priorityP2    = lipgloss.AdaptiveColor{Light: "#268bd2", Dark: "#7aa2f7"}
	heatWarm      = lipgloss.AdaptiveColor{Light: "#b58900", Dark: "#e0af68"}
	heatHot       = lipgloss.AdaptiveColor{Light: "#d70000", Dark: "#f7768e"}
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
	priority map[string]lipgloss.Style
}

func New(r *lipgloss.Renderer) Theme {
	fg := func(c lipgloss.TerminalColor) lipgloss.Style { return r.NewStyle().Foreground(c) }
	return Theme{
		r:      r,
		Text:   r.NewStyle(),
		Accent: fg(accent),
		Dim:    fg(dim),
		Border: fg(border),
		category: map[datamodel.Category]lipgloss.Style{
			datamodel.CategoryTodo:  fg(categoryTodo),
			datamodel.CategoryDoing: fg(categoryDoing),
			datamodel.CategoryDone:  fg(categoryDone),
		},
		priority: map[string]lipgloss.Style{
			"P0": fg(priorityP0),
			"P1": fg(priorityP1),
			"P2": fg(priorityP2),
			"P3": fg(dim),
		},
		Heat: Heat{Warm: fg(heatWarm), Hot: fg(heatHot)},
	}
}

func (t Theme) CategoryStyle(c datamodel.Category) lipgloss.Style {
	if s, ok := t.category[c]; ok {
		return s
	}
	return t.Text
}

func (t Theme) PriorityStyle(p string) lipgloss.Style {
	if s, ok := t.priority[p]; ok {
		return s
	}
	return t.Text
}

func (t Theme) Renderer() *lipgloss.Renderer { return t.r }

func For(w io.Writer, ui datamodel.UI, noColor bool) Theme {
	return New(NewRenderer(w, ui, noColor))
}

func NewRenderer(w io.Writer, ui datamodel.UI, noColor bool) *lipgloss.Renderer {
	r := lipgloss.NewRenderer(w)
	if noColor || os.Getenv("NO_COLOR") != "" {
		r.SetColorProfile(termenv.Ascii)
	}
	r.SetHasDarkBackground(prefersDark(ui.Background))
	return r
}

func prefersDark(bg datamodel.Background) bool {
	return bg != datamodel.BackgroundLight
}
