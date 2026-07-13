package tui

import (
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/termx"
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
)

type iconSet struct {
	mode datamodel.IconMode
}

func (ic iconSet) rich() bool { return ic.mode != datamodel.IconText }

func writerIsTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && termx.IsTerminal(f)
}

func detectIcons(mode datamodel.IconMode, env func(string) string, isTTY bool) iconSet {
	return iconSet{mode: resolveIconMode(mode, env, isTTY)}
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

func (ic iconSet) priorityGlyph(priority string) string {
	if priority == "P0" || priority == "P1" {
		return glyphPriority.pick(ic.mode)
	}
	return ""
}

func (ic iconSet) priorityCell(priority string) string {
	marker := ic.priorityGlyph(priority)
	gutter := lipgloss.Width(glyphPriority.pick(ic.mode))
	return marker + strings.Repeat(" ", gutter-lipgloss.Width(marker))
}
