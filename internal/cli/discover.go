package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/fzfx"
)

type action string

const (
	actionShow action = "show"
	actionEdit action = "edit"
)

func newDiscoverCmd(g *globalFlags) *cobra.Command {
	var act string
	var forceFzf bool
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Interactively pick a ticket or epic (fzf)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sel := action(act)
			if sel != actionShow && sel != actionEdit {
				return errx.User("--action: must be %s or %s, got %q", actionShow, actionEdit, act)
			}
			if g.json {
				return errx.User("discover: --json is not supported").WithHint("discover is an interactive picker; script against `kira list --json` or `kira show --json` instead")
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			cands, err := s.Candidates(cfg)
			if err != nil {
				return err
			}
			if len(cands) == 0 {
				_, _ = fmt.Fprintln(cmd.ErrOrStderr(), msgPrefix, "no items to pick from")
				return nil
			}

			ref, err := pickCandidate(cmd, cands, forceFzf)
			if err != nil || ref == "" {
				return err
			}
			return dispatchAction(cmd, g, s, cfg, sel, ref)
		},
	}
	f := cmd.Flags()
	f.StringVar(&act, "action", string(actionShow), "what to do with the selection (show|edit)")
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
	default:
		return pickNumbered(cmd, cands)
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
		opts.PreviewCmd = shellQuote(exe) + " show {1}"
	}
	selection, err := fzfx.Pick(rows, opts)
	switch {
	case errors.Is(err, fzfx.ErrCancelled):
		return "", nil
	case err != nil:
		return "", errx.Env("%v", err)
	}
	return refFromLine(selection), nil
}

// shellQuote single-quotes s for the POSIX shell fzf runs its preview
// command through, so a path with spaces or shell metacharacters (as from
// os.Executable) survives intact instead of splitting or expanding.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// pickNumbered is the no-fzf fallback: list candidates and read a 1-based
// index from stdin. Unlike fzf's Esc, there is no interactive cancel
// gesture here, so an unreadable or out-of-range pick is a real error
// rather than a silent no-op.
func pickNumbered(cmd *cobra.Command, cands []core.Candidate) (string, error) {
	out := cmd.OutOrStdout()
	for i, c := range cands {
		_, _ = fmt.Fprintf(out, "%d) %s\n", i+1, candidateLine(c))
	}
	_, _ = fmt.Fprint(out, "fzf not found; pick a number: ")
	var n int
	if _, err := fmt.Fscan(cmd.InOrStdin(), &n); err != nil {
		return "", errx.Env("discover: fzf is not on PATH and no selection was read").WithHint("install fzf, or pipe a number 1-N on stdin")
	}
	if n < 1 || n > len(cands) {
		return "", errx.User("discover: %d is out of range (1-%d)", n, len(cands))
	}
	return cands[n-1].Number, nil
}

func dispatchAction(cmd *cobra.Command, g *globalFlags, s *core.Store, cfg *datamodel.Config, act action, ref string) error {
	switch act {
	case actionEdit:
		res, err := s.Edit(cfg, ref, core.EditOpts{})
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), editLine(res))
		return nil
	default:
		res, err := s.Show(cfg, ref, "")
		if err != nil {
			return err
		}
		return printShow(cmd, g, res)
	}
}
