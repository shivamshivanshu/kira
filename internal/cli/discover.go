package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/fzfx"
	"github.com/shivamshivanshu/kira/internal/termx"
)

const (
	actionShow = "show"
	actionEdit = "edit"
)

func newDiscoverCmd(g *globalFlags) *cobra.Command {
	var action string
	var forceFzf bool
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Interactively pick a ticket or epic (fzf)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if action != actionShow && action != actionEdit {
				return fmt.Errorf("--action: must be %s or %s, got %q", actionShow, actionEdit, action)
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			cands, err := s.Candidates()
			if err != nil {
				return err
			}
			if len(cands) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "kira: no items to pick from")
				return nil
			}

			ref, err := pickCandidate(cmd, cands, forceFzf)
			if err != nil || ref == "" {
				return err
			}
			return dispatchAction(cmd, s, cfg, action, ref)
		},
	}
	f := cmd.Flags()
	f.StringVar(&action, "action", actionShow, "what to do with the selection (show|edit)")
	f.BoolVar(&forceFzf, "fzf", false, "require the fzf backend (error if fzf is absent)")
	return cmd
}

// An empty ref with a nil error means the user cancelled: the caller exits 0.
func pickCandidate(cmd *cobra.Command, cands []core.Candidate, forceFzf bool) (string, error) {
	switch {
	case fzfx.Installed():
		return pickFzf(cands)
	case forceFzf:
		return "", errx.Env("discover: --fzf given but fzf is not on PATH")
	case termx.IsTerminal(os.Stdout):
		return pickBubbles(cands)
	default:
		renderCandidateList(cmd.OutOrStdout(), cands)
		return "", nil
	}
}

func candidateLine(c core.Candidate) string {
	return c.Number + "\t" + c.Title
}

func refFromLine(line string) string {
	before, _, _ := strings.Cut(strings.TrimSpace(line), "\t")
	return strings.TrimSpace(before)
}

func pickFzf(cands []core.Candidate) (string, error) {
	rows := make([]string, len(cands))
	for i, c := range cands {
		rows[i] = candidateLine(c)
	}
	opts := fzfx.Options{Prompt: "kira> "}
	if exe, err := os.Executable(); err == nil {
		opts.PreviewCmd = exe + " show {1}"
	}
	selection, aborted, err := fzfx.Pick(rows, opts)
	if err != nil {
		return "", errx.Env("%v", err)
	}
	if aborted {
		return "", nil
	}
	return refFromLine(selection), nil
}

func dispatchAction(cmd *cobra.Command, s *core.Store, cfg *datamodel.Config, action, ref string) error {
	switch action {
	case actionEdit:
		_, err := s.Edit(cfg, ref, core.EditOpts{})
		return err
	default:
		res, err := s.Show(cfg, ref, "")
		if err != nil {
			return err
		}
		renderShow(cmd.OutOrStdout(), res)
		return nil
	}
}

func renderCandidateList(w io.Writer, cands []core.Candidate) {
	for _, c := range cands {
		fmt.Fprintln(w, candidateLine(c))
	}
}

type pickItem struct{ number, title string }

func (i pickItem) Title() string       { return i.number + "  " + i.title }
func (i pickItem) Description() string { return "" }
func (i pickItem) FilterValue() string { return i.number + " " + i.title }

type pickModel struct {
	list   list.Model
	chosen string
}

func (m pickModel) Init() tea.Cmd { return nil }

func (m pickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "enter":
				if it, ok := m.list.SelectedItem().(pickItem); ok {
					m.chosen = it.number
				}
				return m, tea.Quit
			case "q", "ctrl+c", "esc":
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m pickModel) View() string { return m.list.View() }

func pickBubbles(cands []core.Candidate) (string, error) {
	items := make([]list.Item, len(cands))
	for i, c := range cands {
		items[i] = pickItem{number: c.Number, title: c.Title}
	}
	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "kira discover"
	final, err := tea.NewProgram(pickModel{list: l}, tea.WithAltScreen()).Run()
	if err != nil {
		return "", errx.Env("discover picker: %v", err)
	}
	return final.(pickModel).chosen, nil
}
