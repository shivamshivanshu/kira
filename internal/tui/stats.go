package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

const (
	statsEmptyMessage = "No sprint data yet — add a sprint and estimates to see burndown and velocity."
	velocityBarCells  = 12
)

var statsKeys = []KeyBinding{{"j/k", "scroll"}}

func init() { registerScreen(viewStats, func() screen { return newStatsScreen() }) }

type statsScreen struct {
	loaded     bool
	res        *datamodel.StatsResult
	err        error
	scroll     int
	cacheRes   *datamodel.StatsResult
	cacheLines []string
}

func newStatsScreen() *statsScreen { return &statsScreen{} }

func (s *statsScreen) keys() []KeyBinding { return statsKeys }

func (s *statsScreen) update(m *model, key string) tea.Cmd {
	switch key {
	case "j", "down":
		s.scroll++
	case "k", "up":
		s.scroll--
	}
	return nil
}

func (s *statsScreen) back(m *model) bool { return false }

func (s *statsScreen) focusItem(m *model, id string) {}

func (s *statsScreen) setResult(res *datamodel.StatsResult) {
	s.res = res
	s.loaded = true
}

func (s *statsScreen) invalidate() { s.loaded = false }

func (s *statsScreen) ensure(m *model) {
	if s.loaded || m.store == nil {
		return
	}
	s.res, s.err = loadStats(m.store, m.cfg)
	s.loaded = true
}

func (s *statsScreen) view(m *model, width, height int) string {
	s.ensure(m)
	if s.err != nil {
		return centered(m.theme, width, height, m.theme.Dim.Render("cannot load stats: "+s.err.Error()))
	}
	lines := s.contentLines(m.theme, m.icons.rich())
	if len(lines) == 0 {
		return centered(m.theme, width, height, m.theme.Dim.Render(statsEmptyMessage))
	}
	return renderScrollable(m.theme, lines, &s.scroll, width, height)
}

func (s *statsScreen) contentLines(t theme.Theme, rich bool) []string {
	if s.cacheRes != s.res {
		s.cacheRes = s.res
		s.cacheLines = statsLines(t, rich, s.res)
	}
	return s.cacheLines
}

func loadStats(store *core.Store, cfg *datamodel.Config) (*datamodel.StatsResult, error) {
	if res, err := store.Stats(cfg, core.StatsOpts{Sprint: "active", Velocity: true}); err == nil {
		return res, nil
	}
	return store.Stats(cfg, core.StatsOpts{Velocity: true})
}

func statsLines(t theme.Theme, rich bool, res *datamodel.StatsResult) []string {
	if res == nil {
		return nil
	}
	var lines []string
	if b := res.Burndown; b != nil && len(b.Days) > 0 {
		lines = append(lines, burndownLines(t, rich, b)...)
	}
	if v := res.Velocity; v != nil && len(v.Sprints) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, velocityLines(t, rich, v)...)
	}
	return lines
}

func burndownLines(t theme.Theme, rich bool, b *datamodel.Burndown) []string {
	remaining := make([]float64, len(b.Days))
	ideal := make([]float64, len(b.Days))
	for i, d := range b.Days {
		remaining[i], ideal[i] = d.Remaining, d.Ideal
	}
	last := codec.EmitFloat(b.Days[len(b.Days)-1].Remaining)
	lines := []string{
		t.Accent.Render("Burndown") + "  " + b.Sprint + "  " + b.Start + " -> " + b.End + " (" + b.Unit + ")",
		sparkRow(t, "remaining", sparkline(remaining, rich), last),
		sparkRow(t, "ideal", sparkline(ideal, rich), ""),
	}
	if b.Unestimated > 0 {
		lines = append(lines, t.Dim.Render(fmt.Sprintf("  %d unestimated item(s) burn nothing", b.Unestimated)))
	}
	if b.DegradedN > 0 {
		lines = append(lines, t.Dim.Render(fmt.Sprintf("  %d item(s) with lossy history", b.DegradedN)))
	}
	return lines
}

func velocityLines(t theme.Theme, rich bool, v *datamodel.Velocity) []string {
	var maxV float64
	for _, sp := range v.Sprints {
		if sp.Completed > maxV {
			maxV = sp.Completed
		}
	}
	lines := []string{t.Accent.Render("Velocity") + " (" + v.Unit + ")"}
	for _, sp := range v.Sprints {
		row := t.Dim.Render(fmt.Sprintf("  %-10s", sp.Key)) + hbar(sp.Completed, maxV, velocityBarCells, rich) + "  " + codec.EmitFloat(sp.Completed)
		lines = append(lines, row)
	}
	return append(lines, t.Dim.Render("  trailing-3  ")+codec.EmitFloat(v.Trailing3))
}

func sparkRow(t theme.Theme, label, spark, suffix string) string {
	row := t.Dim.Render(fmt.Sprintf("  %-10s", label)) + spark
	if suffix != "" {
		row += "  " + suffix
	}
	return row
}
