package tui

import (
	"io"
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/termx"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

type glyph struct {
	nerd  string
	emoji string
	ascii string
}

func (g glyph) pick(mode datamodel.IconMode) string {
	switch mode {
	case datamodel.IconNerd:
		return g.nerd
	case datamodel.IconEmoji:
		return g.emoji
	default:
		return g.ascii
	}
}

var (
	glyphEpic     = glyph{"", "📦", "[E]"}
	glyphTicket   = glyph{"", "🎫", "[T]"}
	glyphTodo     = glyph{"", "⬜", "[ ]"}
	glyphDoing    = glyph{"", "🔄", "[~]"}
	glyphDone     = glyph{"", "✅", "[x]"}
	glyphDropped  = glyph{"", "🚫", "[-]"}
	glyphPriority = glyph{"", "❗", "!"}
	glyphOverdue  = glyph{"", "⚠", "!"}
)

type iconSet struct {
	mode       datamodel.IconMode
	priorities []string
}

func (ic iconSet) rich() bool { return ic.mode != datamodel.IconText }

func writerIsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && termx.IsTerminal(f)
}

func detectIcons(mode datamodel.IconMode, priorities []string, env func(string) string, isTTY bool) iconSet {
	return iconSet{mode: resolveIconMode(mode, env, isTTY), priorities: priorities}
}

func resolveIconMode(mode datamodel.IconMode, env func(string) string, isTTY bool) datamodel.IconMode {
	if canonical, ok := canonicalIconMode(mode); ok {
		return canonical
	}
	if canonical, ok := canonicalIconMode(datamodel.IconMode(env("KIRA_ICONS"))); ok {
		return canonical
	}
	return autoIconMode(env, isTTY)
}

func canonicalIconMode(mode datamodel.IconMode) (datamodel.IconMode, bool) {
	switch mode {
	case datamodel.IconNerd, datamodel.IconAlways:
		return datamodel.IconNerd, true
	case datamodel.IconEmoji:
		return datamodel.IconEmoji, true
	case datamodel.IconText, datamodel.IconNever:
		return datamodel.IconText, true
	}
	return "", false
}

func autoIconMode(env func(string) string, isTTY bool) datamodel.IconMode {
	if !isTTY || terminalClearlyIncapable(env) {
		return datamodel.IconText
	}
	return datamodel.IconNerd
}

func terminalClearlyIncapable(env func(string) string) bool {
	if env("TERM") == "dumb" {
		return true
	}
	return firstLocale(env) != "" && !utf8Locale(env)
}

func firstLocale(env func(string) string) string {
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := env(key); v != "" {
			return v
		}
	}
	return ""
}

func utf8Locale(env func(string) string) bool {
	loc := strings.ToLower(firstLocale(env))
	return strings.Contains(loc, "utf-8") || strings.Contains(loc, "utf8")
}

func osEnv(key string) string { return os.Getenv(key) }

func (ic iconSet) typeGlyph(typ string) string {
	if typ == datamodel.TypeEpic {
		return glyphEpic.pick(ic.mode)
	}
	return glyphTicket.pick(ic.mode)
}

func (ic iconSet) categoryGlyph(cat datamodel.Category, resolution *string) string {
	switch cat {
	case datamodel.CategoryDoing:
		return glyphDoing.pick(ic.mode)
	case datamodel.CategoryDone:
		if resolution != nil && *resolution == datamodel.ResolutionDropped {
			return glyphDropped.pick(ic.mode)
		}
		return glyphDone.pick(ic.mode)
	default:
		return glyphTodo.pick(ic.mode)
	}
}

func (ic iconSet) overdueGlyph() string { return glyphOverdue.pick(ic.mode) }

func (ic iconSet) priorityTier(value string) int {
	if value == "" {
		return -1
	}
	return slices.Index(ic.priorities, value)
}

func priorityMarks(tier int) int {
	switch {
	case tier < 0:
		return 0
	case tier == 0:
		return 3
	case tier == 1:
		return 2
	default:
		return 1
	}
}

func (ic iconSet) priorityCell(value string) string {
	unit := glyphPriority.pick(ic.mode)
	gutter := lipgloss.Width(unit)
	if len(ic.priorities) > 0 {
		gutter *= 3
	}
	cell := strings.Repeat(unit, priorityMarks(ic.priorityTier(value)))
	return cell + strings.Repeat(" ", gutter-lipgloss.Width(cell))
}

func priorityHue(t theme.Theme, tier int) lipgloss.Style {
	switch tier {
	case 0:
		return t.Heat.Hot
	case 1:
		return t.Heat.Warm
	default:
		return t.Dim
	}
}
