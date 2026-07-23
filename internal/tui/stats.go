package tui

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/showfmt"
	"github.com/shivamshivanshu/kira/internal/statsfmt"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

const statsEmptyMessage = "No metrics yet — create and move some tickets to see completion, cycle time, and throughput."

var statsKeys = []KeyBinding{
	{"j/k", "scroll"},
	{"gg/G", "top/bottom"},
	{"^d/^u", "half-page"},
}

type statsLoadState int

const (
	statsNotLoaded statsLoadState = iota
	statsPending
	statsReady
)

type statsScreen struct {
	state statsLoadState
	res   *datamodel.StatsResult
	err   error
	scroller
	cacheRes   *datamodel.StatsResult
	cacheLines []string
}

func newStatsScreen() *statsScreen { return &statsScreen{} }

func (s *statsScreen) keys() []KeyBinding { return statsKeys }

func (s *statsScreen) update(m *model, key string) tea.Cmd {
	s.scroller.update(key, m.mainHeight()/2)
	return nil
}

func (s *statsScreen) back(_ *model) bool { return false }

func (s *statsScreen) focusItem(_ *model, _ string) {}

func (s *statsScreen) focusedItem() (showfmt.Item, bool) { return showfmt.Item{}, false }

func (s *statsScreen) settle(_ *model) {}

func (s *statsScreen) invalidate() { s.state = statsNotLoaded }

// activate dispatches an async reload through the model's command queue at
// most once per invalidation; the result lands later via statsLoadedMsg.
func (s *statsScreen) activate(m *model) tea.Cmd {
	if s.state != statsNotLoaded || m.store == nil {
		return nil
	}
	s.state = statsPending
	return m.request(statsLoadCmd(m.store, m.cfg))
}

func (s *statsScreen) applyLoaded(msg statsLoadedMsg) {
	s.res, s.err = msg.res, msg.err
	s.state = statsReady
}

func (s *statsScreen) view(m *model, width, height int) string {
	if s.err != nil {
		return centered(m.theme, width, height, m.theme.Heat.Hot.Render("cannot load stats: "+firstNonEmptyLine(s.err.Error())))
	}
	if s.state == statsPending {
		return renderLoading(m.theme, width, height)
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
	res, err := store.Stats(cfg, core.StatsOpts{Sprint: "active"})
	if errors.Is(err, core.ErrNoActiveSprint) {
		return store.Stats(cfg, core.StatsOpts{})
	}
	return res, err
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
	lines = append(lines, t.Accent.Render("Completion")+"  "+statsfmt.CompletionLine(c))
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
		lines = append(lines, statLabel(t, "reopens")+strings.Join(r.Items, ", "))
	}
	return lines
}

const statLabelGutter = 12

func statLabel(t theme.Theme, label string) string {
	return t.Dim.Render(fmt.Sprintf("  %-*s", statLabelGutter, label))
}

func percentileLine(t theme.Theme, label string, p *datamodel.Percentiles) string {
	if p == nil || p.N == 0 {
		return statLabel(t, label) + "no data"
	}
	return statLabel(t, label) + statsfmt.PercentileLine(p)
}

func sparkRow(t theme.Theme, label, spark, suffix string) string {
	row := statLabel(t, label) + spark
	if suffix != "" {
		row += "  " + suffix
	}
	return row
}
