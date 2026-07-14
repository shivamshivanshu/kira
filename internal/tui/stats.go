package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

const statsEmptyMessage = "No metrics yet — create and move some tickets to see completion, cycle time, and throughput."

var statsKeys = []KeyBinding{
	{"j/k", "scroll"},
	{"gg/G", "top/bottom"},
	{"^d/^u", "half-page"},
}

func init() { registerScreen(viewStats, func() screen { return newStatsScreen() }) }

type statsScreen struct {
	loaded     bool
	res        *datamodel.StatsResult
	err        error
	scroll     int
	pendingG   bool
	cacheRes   *datamodel.StatsResult
	cacheLines []string
}

func newStatsScreen() *statsScreen { return &statsScreen{} }

func (s *statsScreen) keys() []KeyBinding { return statsKeys }

func (s *statsScreen) update(m *model, key string) tea.Cmd {
	if s.pendingG {
		s.pendingG = false
		if key == "g" {
			s.scroll = 0
		}
		return nil
	}
	switch key {
	case "j", "down":
		s.scroll++
	case "k", "up":
		s.scroll--
	case "ctrl+d":
		s.scroll += m.mainHeight() / 2
	case "ctrl+u":
		s.scroll -= m.mainHeight() / 2
	case "g":
		s.pendingG = true
	case "G":
		s.scroll = 1 << 30
	}
	return nil
}

func (s *statsScreen) back(m *model) bool { return false }

func (s *statsScreen) focusItem(m *model, id string) {}

func (s *statsScreen) settle(m *model) {}

func (s *statsScreen) setResult(res *datamodel.StatsResult) {
	s.res = res
	s.loaded = true
}

func (s *statsScreen) invalidate() { s.loaded = false }

func (s *statsScreen) ensure(m *model) {
	if s.loaded || m.store == nil || m.busy {
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
	if res, err := store.Stats(cfg, core.StatsOpts{Sprint: "active"}); err == nil {
		return res, nil
	}
	return store.Stats(cfg, core.StatsOpts{})
}

func statsLines(t theme.Theme, rich bool, res *datamodel.StatsResult) []string {
	if res == nil || res.Completion == nil || res.Completion.Total == 0 {
		return nil
	}
	var lines []string
	if s := res.Scope; s != nil && s.Sprint != "" {
		lines = append(lines, t.Accent.Render("Sprint")+"  "+s.Sprint)
	}
	c := res.Completion
	head := fmt.Sprintf("%d/%d done (%.0f%%)", c.Done, c.Total, c.Pct*100)
	if c.Dropped > 0 {
		head += fmt.Sprintf(", %d dropped", c.Dropped)
	}
	lines = append(lines, t.Accent.Render("Completion")+"  "+head)
	lines = append(lines, percentileLine(t, "cycle time", res.CycleTime))
	lines = append(lines, percentileLine(t, "lead time", res.LeadTime))
	if len(res.Throughput) > 0 {
		vals := make([]float64, len(res.Throughput))
		total := 0
		for i, n := range res.Throughput {
			vals[i] = float64(n)
			total += n
		}
		lines = append(lines, sparkRow(t, "throughput", sparkline(vals, rich), fmt.Sprintf("%d in %dw", total, len(res.Throughput))))
	}
	if r := res.Reopens; r != nil && r.Count > 0 {
		lines = append(lines, t.Dim.Render(fmt.Sprintf("  %-12s", "reopens"))+strings.Join(r.Items, ", "))
	}
	return lines
}

func percentileLine(t theme.Theme, label string, p *datamodel.Percentiles) string {
	if p == nil || p.N == 0 {
		return t.Dim.Render(fmt.Sprintf("  %-12s", label)) + "no data"
	}
	return t.Dim.Render(fmt.Sprintf("  %-12s", label)) +
		fmt.Sprintf("p50 %s  p90 %s  n=%d", codec.EmitFloat(p.P50), codec.EmitFloat(p.P90), p.N)
}

func sparkRow(t theme.Theme, label, spark, suffix string) string {
	row := t.Dim.Render(fmt.Sprintf("  %-12s", label)) + spark
	if suffix != "" {
		row += "  " + suffix
	}
	return row
}
