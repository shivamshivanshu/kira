package tui

import (
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

type crashMsg struct {
	value any
	stack []byte
}

type tickMsg struct{}

type treeData struct {
	nodes    []datamodel.TreeNode
	fields   map[string]datamodel.ListItem
	progress map[string]datamodel.EpicProgress
}

type treeLoadedMsg struct {
	data treeData
	err  error
}

func safeCmd(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() (msg tea.Msg) {
		defer func() {
			if r := recover(); r != nil {
				msg = crashMsg{value: r, stack: debug.Stack()}
			}
		}()
		return cmd()
	}
}

func refreshCmd(store *core.Store, cfg *datamodel.Config, filter string) tea.Cmd {
	return safeCmd(func() tea.Msg {
		data, err := loadFilteredTree(store, cfg, filter)
		return treeLoadedMsg{data: data, err: err}
	})
}

func loadDetail(store *core.Store, cfg *datamodel.Config, id string) (*datamodel.ShowResult, error) {
	if id == "" {
		return nil, nil
	}
	return store.Show(cfg, id, "")
}

type statsLoadedMsg struct {
	res *datamodel.StatsResult
	err error
}

func statsLoadCmd(store *core.Store, cfg *datamodel.Config) tea.Cmd {
	return safeCmd(func() tea.Msg {
		res, err := loadStats(store, cfg)
		return statsLoadedMsg{res: res, err: err}
	})
}

type boardMovedMsg struct {
	res    *datamodel.MoveResult
	board  *datamodel.BoardResult
	err    error
	cardID string
}

func boardMoveCmd(store *core.Store, cfg *datamodel.Config, cardID, to string) tea.Cmd {
	return safeCmd(func() tea.Msg {
		res, err := store.Move(cfg, cardID, to, core.MoveOpts{})
		if err != nil {
			return boardMovedMsg{err: err, cardID: cardID}
		}
		board, err := store.Board(cfg, core.BoardOpts{})
		return boardMovedMsg{res: res, board: board, err: err, cardID: cardID}
	})
}
