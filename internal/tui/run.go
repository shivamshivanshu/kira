package tui

import (
	"errors"
	"io"
	"os"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/clipx"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/tui/theme"
)

type View = view

const (
	ViewTree  = viewTree
	ViewBoard = viewBoard
	ViewStats = viewStats
)

type Options struct {
	NoColor     bool
	Input       io.Reader
	Output      io.Writer
	InjectPanic bool
	RunCommand  func([]string) (string, error)
	InitialView View
}

func Run(store *core.Store, cfg *datamodel.Config, opts Options) error {
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}
	return guardRun(store.Root(), os.Stderr, func() error {
		m := newModel(store, cfg, theme.For(out, cfg.UI, opts.NoColor), detectIcons(cfg.UI.Icons, osEnv, writerIsTTY(out)), opts.InjectPanic)
		m.bar.run = opts.RunCommand
		m.view = opts.InitialView
		m.clip = clipx.System(out)
		if !opts.InjectPanic {
			if data, e := loadTreeData(store, cfg); e != nil {
				m.loadErr = e
			} else if ts, ok := m.screens[viewTree].(*treeScreen); ok {
				ts.setData(&m, data)
			}
		}

		final, runErr := tea.NewProgram(m, programOptions(opts, out)...).Run()
		if fm, ok := final.(model); ok && fm.crash != nil {
			return handleCrash(store.Root(), *fm.crash, os.Stderr)
		}
		if runErr != nil && errors.Is(runErr, tea.ErrProgramPanic) {
			return handleCrash(store.Root(), crashInfo{value: runErr, stack: debug.Stack()}, os.Stderr)
		}
		return runErr
	})
}

func guardRun(root string, stderr io.Writer, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handleCrash(root, crashInfo{value: r, stack: debug.Stack()}, stderr)
		}
	}()
	return fn()
}

func programOptions(opts Options, out io.Writer) []tea.ProgramOption {
	if opts.InjectPanic {
		in := opts.Input
		if in == nil {
			in = strings.NewReader("")
		}
		return []tea.ProgramOption{tea.WithInput(in), tea.WithOutput(io.Discard)}
	}
	po := []tea.ProgramOption{tea.WithAltScreen(), tea.WithOutput(out)}
	if opts.Input != nil {
		po = append(po, tea.WithInput(opts.Input))
	}
	return po
}
