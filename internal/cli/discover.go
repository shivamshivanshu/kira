package cli

import (
	"errors"
	"fmt"
	"io"
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
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			cands, err := s.Candidates(cfg)
			if err != nil {
				return err
			}
			if len(cands) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), msgPrefix, "no items to pick from")
				return nil
			}

			ref, err := pickCandidate(cmd, cands, forceFzf)
			if err != nil || ref == "" {
				return err
			}
			return dispatchAction(cmd, s, cfg, sel, ref)
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
	selection, err := fzfx.Pick(rows, opts)
	switch {
	case errors.Is(err, fzfx.ErrCancelled):
		return "", nil
	case err != nil:
		return "", errx.Env("%v", err)
	}
	return refFromLine(selection), nil
}

func dispatchAction(cmd *cobra.Command, s *core.Store, cfg *datamodel.Config, act action, ref string) error {
	switch act {
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
