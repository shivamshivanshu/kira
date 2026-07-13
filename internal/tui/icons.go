package tui

import (
	"os"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

type glyph struct {
	nerd  string
	ascii string
}

func (g glyph) pick(nerd bool) string {
	if nerd {
		return g.nerd
	}
	return g.ascii
}

var (
	glyphEpic     = glyph{"", "[E]"}
	glyphTicket   = glyph{"", "[T]"}
	glyphTodo     = glyph{"", "[ ]"}
	glyphDoing    = glyph{"", "[~]"}
	glyphDone     = glyph{"", "[x]"}
	glyphDropped  = glyph{"", "[-]"}
	glyphPriority = glyph{"", "!"}
)

type iconSet struct {
	nerd bool
}

func detectIcons(mode datamodel.IconMode, env func(string) string) iconSet {
	return iconSet{nerd: resolveNerd(mode, env)}
}

func resolveNerd(mode datamodel.IconMode, env func(string) string) bool {
	switch mode {
	case datamodel.IconAlways:
		return true
	case datamodel.IconNever:
		return false
	}
	switch env("KIRA_ICONS") {
	case "always":
		return true
	case "never":
		return false
	}
	return nerdLikelyTerminal(env)
}

func nerdLikelyTerminal(env func(string) string) bool {
	switch env("TERM_PROGRAM") {
	case "WezTerm", "kitty", "iTerm.app":
		return true
	}
	term := env("TERM")
	return strings.Contains(term, "kitty") || strings.Contains(term, "alacritty")
}

func osEnv(key string) string { return os.Getenv(key) }

func (ic iconSet) typeGlyph(typ string) string {
	if typ == datamodel.TypeEpic {
		return glyphEpic.pick(ic.nerd)
	}
	return glyphTicket.pick(ic.nerd)
}

func (ic iconSet) categoryGlyph(cat datamodel.Category, resolution *string) string {
	switch cat {
	case datamodel.CategoryDoing:
		return glyphDoing.pick(ic.nerd)
	case datamodel.CategoryDone:
		if resolution != nil && *resolution == datamodel.ResolutionDropped {
			return glyphDropped.pick(ic.nerd)
		}
		return glyphDone.pick(ic.nerd)
	default:
		return glyphTodo.pick(ic.nerd)
	}
}

func (ic iconSet) priorityGlyph(priority string) string {
	if priority == "P0" || priority == "P1" {
		return glyphPriority.pick(ic.nerd)
	}
	return ""
}
