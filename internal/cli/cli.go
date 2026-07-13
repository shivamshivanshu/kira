// Package cli wires the kira cobra command tree; every command is a thin argv/flag adapter over internal/core.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/tui"
)

func Main() int {
	root, g := newRootCmd()
	if err := root.Execute(); err != nil {
		return renderError(os.Stderr, err, g.json)
	}
	return 0
}

func renderError(w io.Writer, err error, jsonMode bool) int {
	var crash *tui.CrashError
	if errors.As(err, &crash) {
		return errx.ExitCrash
	}
	code := errx.ExitUser
	hint := ""
	var ce *errx.Error
	if errors.As(err, &ce) {
		code, hint = ce.Code, ce.Hint
	}
	if jsonMode {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(struct {
			Error string `json:"error"`
			Hint  string `json:"hint"`
			Code  int    `json:"code"`
		}{err.Error(), hint, code})
		return code
	}
	fmt.Fprintln(w, "kira:", err)
	if hint != "" {
		fmt.Fprintln(w, "  hint:", hint)
	}
	return code
}

type globalFlags struct {
	json    bool
	noColor bool
	chdir   string
	quiet   bool
}

func newRootCmd() (*cobra.Command, *globalFlags) {
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
		newAutomationCmd(g),
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
		newConfigCmd(g),
	)
	attachCompletions(root, g)
	return root, g
}

func openStore(g *globalFlags) (*core.Store, *datamodel.Config, error) {
	s, err := core.Discover(g.chdir, core.WithPrompter(terminalPrompter{}))
	if err != nil {
		return nil, nil, err
	}
	cfg, err := s.Config()
	if err != nil {
		return nil, nil, err
	}
	return s, cfg, nil
}
