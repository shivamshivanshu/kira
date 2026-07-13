package tui

import tea "github.com/charmbracelet/bubbletea"

type screen interface {
	keys() []KeyBinding
	update(m *model, key string) tea.Cmd
	view(m *model, width, height int) string
	back(m *model) bool
	focusItem(m *model, id string)
}

var screenFactories = map[view]func() screen{}

func registerScreen(v view, factory func() screen) { screenFactories[v] = factory }

func buildScreens() map[view]screen {
	screens := make(map[view]screen, len(screenFactories))
	for v, factory := range screenFactories {
		screens[v] = factory()
	}
	return screens
}
