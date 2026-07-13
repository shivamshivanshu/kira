// Package cli wires the kira cobra command tree; every command is a thin argv/flag adapter over internal/core.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/tui"
)

func Main() int {
	if err := newRootCmd().Execute(); err != nil {
		var crash *tui.CrashError
		if errors.As(err, &crash) {
			return errx.ExitCrash
		}
		fmt.Fprintln(os.Stderr, "kira:", err)
		var ce *errx.Error
		if errors.As(err, &ce) {
			return ce.Code
		}
		return 1
	}
	return 0
}

type globalFlags struct {
	json    bool
	noColor bool
	chdir   string
	quiet   bool
}

func newRootCmd() *cobra.Command {
	g := &globalFlags{}
	root := &cobra.Command{
		Use:           "kira",
		Short:         "kira — a git-native, terminal-first ticket tracker",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(g, false)
		},
	}
	pf := root.PersistentFlags()
	pf.BoolVar(&g.json, "json", false, "emit machine-readable JSON on stdout")
	pf.BoolVar(&g.noColor, "no-color", false, "disable ANSI color in human output")
	pf.StringVarP(&g.chdir, "C", "C", "", "run as if invoked from `path`")
	pf.BoolVar(&g.quiet, "quiet", false, "suppress non-essential human output")

	root.AddCommand(
		newTUICmd(g),
		newVersionCmd(),
		newInitCmd(g),
		newCreateCmd(g),
		newShowCmd(g),
		newEditCmd(g),
		newMoveCmd(g),
		newAssignCmd(g),
		newLinkCmd(g),
		newCommentCmd(g),
		newListCmd(g),
		newQueryCmd(g),
		newFilterCmd(g),
		newTreeCmd(g),
		newFindCmd(g),
		newDiscoverCmd(g),
		newHooksCmd(g),
		newWorkonCmd(g),
		newSyncCmd(g),
		newSprintCmd(g),
		newStatsCmd(g),
		newIndexCmd(g),
		newLogCmd(g),
		newBlameCmd(g),
		newDoctorCmd(g),
		newValidateCmd(g),
		newMergeFileCmd(g),
		newResolveCmd(g),
		newDiffCmd(g),
		newBoardCmd(g),
	)
	return root
}

func openStore(g *globalFlags) (*core.Store, *datamodel.Config, error) {
	s, err := core.Discover(g.chdir)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := s.Config()
	if err != nil {
		return nil, nil, err
	}
	return s, cfg, nil
}
