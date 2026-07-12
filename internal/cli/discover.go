package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/core"
)

const (
	actionShow = "show"
	actionEdit = "edit"
)

// newDiscoverCmd builds `kira discover`: an interactive fuzzy picker over
// items. fzf is the primary backend (with a `kira show` preview); absent, it
// falls back to an in-process bubbles picker on a tty, or a plain sorted list
// when stdout is not a terminal (docs/design/04-cli.md discover). No --json:
// this is a selector, not a scriptable read — scripts use `list`/`query`.
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
				return err // err==nil,ref=="" means the user cancelled: exit 0
			}
			return dispatchAction(cmd, s, cfg, action, ref)
		},
	}
	f := cmd.Flags()
	f.StringVar(&action, "action", actionShow, "what to do with the selection (show|edit)")
	f.BoolVar(&forceFzf, "fzf", false, "require the fzf backend (error if fzf is absent)")
	return cmd
}

// pickCandidate resolves the selection ref (a display number) from the chosen
// backend. An empty ref with a nil error means the user cancelled.
func pickCandidate(cmd *cobra.Command, cands []core.Candidate, forceFzf bool) (string, error) {
	switch {
	case core.HaveFzf():
		return pickFzf(cands)
	case forceFzf:
		return "", core.NewEnvError("discover: --fzf given but fzf is not on PATH")
	case core.IsTerminal(os.Stdout):
		return pickBubbles(cands)
	default:
		renderCandidateList(cmd.OutOrStdout(), cands)
		return "", nil
	}
}

// candidateLine is the one-line-per-item picker row. The display number is the
// first whitespace field, so fzf's `{1}` placeholder and our own parse both
// recover it.
func candidateLine(c core.Candidate) string {
	return c.Number + "\t" + c.Title
}

func refFromLine(line string) string {
	return strings.TrimSpace(strings.SplitN(strings.TrimSpace(line), "\t", 2)[0])
}

// pickFzf feeds the candidate lines to fzf and reads back the selected line.
// fzf reads keys from /dev/tty itself, so it works even when kira's stdout is
// piped. A non-zero fzf exit (Esc/no match) means no selection, not an error.
func pickFzf(cands []core.Candidate) (string, error) {
	var stdin strings.Builder
	for _, c := range cands {
		stdin.WriteString(candidateLine(c))
		stdin.WriteByte('\n')
	}
	args := []string{"--with-nth", "1..", "--prompt", "kira> "}
	if exe, err := os.Executable(); err == nil {
		args = append(args, "--preview", exe+" show {1}")
	}
	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(stdin.String())
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "", nil // cancelled / no match
		}
		return "", core.NewEnvError("running fzf: %v", err)
	}
	return refFromLine(string(out)), nil
}

// dispatchAction runs the selected item through the chosen verb, reusing the
// same core services the direct commands call.
func dispatchAction(cmd *cobra.Command, s *core.Store, cfg *config.Config, action, ref string) error {
	switch action {
	case actionEdit:
		_, err := s.Edit(cfg, ref, core.EditOpts{})
		return err
	default: // show
		res, err := s.Show(cfg, ref)
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

// --- bubbles fallback picker (interactive, tty-only) ---

// pickItem is one row in the bubbles list; it satisfies list.DefaultItem.
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
		// While the filter input is focused, keys belong to it; only act on
		// enter/quit once the list is in its normal navigation state.
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
		return "", core.NewEnvError("discover picker: %v", err)
	}
	return final.(pickModel).chosen, nil
}
