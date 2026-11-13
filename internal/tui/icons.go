package tui

import (
	"os"
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/shivamshivanshu/kira/internal/datamodel"
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
	glyphTodo     = glyph{"", "⬜", "[ ]"}
	glyphDoing    = glyph{"", "🔄", "[~]"}
	glyphDone     = glyph{"", "✅", "[x]"}
	glyphDropped  = glyph{"", "🚫", "[-]"}
	glyphPriority = glyph{"", "❗", "!"}
	glyphOverdue  = glyph{"", "⚠", "!"}
)

type iconSet struct {
	mode datamodel.IconMode
	// typeIcons is each item type's glyph, keyed by type. Only the tree draws
	// a per-type glyph, so the plain board renderer leaves this unset.
	typeIcons map[string]string
}

// itemTypeIcons are nerd-font glyphs with no per-mode variants, so ui.icons
// selects state and priority glyphs but never the type glyph.
var itemTypeIcons = map[string]string{
	datamodel.TypeTicket: "\uf41b",
	datamodel.TypeEpic:   "\uf45d",
}

func (ic iconSet) rich() bool { return ic.mode != datamodel.IconText }

func detectIcons(mode datamodel.IconMode, env func(string) string, isTTY bool) iconSet {
	return iconSet{mode: resolveIconMode(mode, env, isTTY)}
}

func resolveIconMode(mode datamodel.IconMode, env func(string) string, isTTY bool) datamodel.IconMode {
	if isExplicitIconMode(mode) {
		return mode
	}
	if envMode := datamodel.IconMode(env("KIRA_ICONS")); isExplicitIconMode(envMode) {
		return envMode
	}
	return autoIconMode(env, isTTY)
}

func isExplicitIconMode(mode datamodel.IconMode) bool {
	switch mode {
	case datamodel.IconNerd, datamodel.IconEmoji, datamodel.IconText:
		return true
	}
	return false
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

// typeGlyph draws the type's glyph in every icon mode.
func (ic iconSet) typeGlyph(typ string) string {
	return ic.typeIcons[typ]
}

// padToWidth trails spaces so a narrow or absent glyph still fills its cell;
// fitWidth only truncates.
func padToWidth(s string, width int) string {
	if pad := width - lipgloss.Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

func (ic iconSet) categoryGlyph(cat datamodel.Category, resolution *string) string {
	switch cat {
	case datamodel.CategoryDoing:
		return glyphDoing.pick(ic.mode)
	case datamodel.CategoryDone:
		if datamodel.IsDropped(resolution) {
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
	return slices.Index(datamodel.Priorities, value)
}

const maxPriorityMarks = 3

func priorityMarks(tier int) int {
	switch {
	case tier < 0:
		return 0
	case tier == 0:
		return maxPriorityMarks
	case tier == 1:
		return 2
	default:
		return 1
	}
}

func (ic iconSet) priorityCell(value string) string {
	unit := glyphPriority.pick(ic.mode)
	gutter := lipgloss.Width(unit) * maxPriorityMarks
	cell := strings.Repeat(unit, priorityMarks(ic.priorityTier(value)))
	return padToWidth(cell, gutter)
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
