package tui

import (
	"io"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/editorx"
)

type editorDoneMsg struct{ err error }

type editorSession struct {
	run   func(stdio editorx.Stdio) error
	stdio editorx.Stdio
}

func (s *editorSession) Run() error            { return s.run(s.stdio) }
func (s *editorSession) SetStdin(r io.Reader)  { s.stdio.In = r }
func (s *editorSession) SetStdout(w io.Writer) { s.stdio.Out = w }
func (s *editorSession) SetStderr(w io.Writer) { s.stdio.Err = w }

func editItemCmd(store *core.Store, cfg *datamodel.Config, ref string) (tea.Cmd, error) {
	if _, err := editorx.Command(cfg.UI.Editor); err != nil {
		return nil, err
	}
	session := &editorSession{run: func(stdio editorx.Stdio) error {
		_, err := store.Edit(cfg, ref, core.EditOpts{Stdio: stdio})
		return err
	}}
	return tea.Exec(session, func(err error) tea.Msg { return editorDoneMsg{err: err} }), nil
}

func viewItemCmd(store *core.Store, cfg *datamodel.Config, ref string) (tea.Cmd, error) {
	view, path, err := viewProcess(store, cfg, ref)
	if err != nil {
		return nil, err
	}
	return tea.ExecProcess(view, func(err error) tea.Msg {
		os.Remove(path)
		return editorDoneMsg{err: err}
	}), nil
}

func viewProcess(store *core.Store, cfg *datamodel.Config, ref string) (*exec.Cmd, string, error) {
	_, content, err := store.ResolveItemFile(cfg, ref)
	if err != nil {
		return nil, "", err
	}
	path, err := readonlyCopy(content)
	if err != nil {
		return nil, "", err
	}
	view, err := editorx.View(cfg.UI.Editor, path)
	if err != nil {
		os.Remove(path)
		return nil, "", err
	}
	return view, path, nil
}

func readonlyCopy(content string) (string, error) {
	tmp, err := os.CreateTemp("", "kira-view-*.md")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	_, err = tmp.WriteString(content)
	if err == nil {
		err = tmp.Chmod(0o400)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}
