// Package cli wires the kira cobra command tree; every command is a thin argv/flag adapter over internal/core.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/tui"
)

const (
	msgPrefix  = "kira:"
	msgNoItems = "no items"
)

// Main is the CLI entry point invoked by cmd/kira.
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
		return int(errx.ExitCrash)
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
		}{err.Error(), hint, int(code)})
		return int(code)
	}
	_, _ = fmt.Fprintln(w, msgPrefix, err)
	if hint != "" {
		_, _ = fmt.Fprintln(w, "  hint:", hint)
	}
	return int(code)
}

type globalFlags struct {
	json           bool
	noColor        bool
	chdir          string
	quiet          bool
	nonInteractive bool
}

func registerGlobalFlags(fs *pflag.FlagSet, g *globalFlags) {
	fs.BoolVar(&g.json, "json", false, "emit machine-readable JSON on stdout")
	fs.BoolVar(&g.noColor, "no-color", false, "disable ANSI color in human output")
	fs.StringVarP(&g.chdir, "C", "C", "", "run as if invoked from `path`")
	fs.BoolVar(&g.quiet, "quiet", false, "suppress non-essential human output")
	fs.BoolVar(&g.nonInteractive, "non-interactive", false, "")
	_ = fs.MarkHidden("non-interactive")
}

func newRootCmd() (*cobra.Command, *globalFlags) {
	g := &globalFlags{}
	root := &cobra.Command{
		Use:           "kira",
		Short:         "kira — a git-native, terminal-first ticket tracker",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTUI(cmd, g, false, true)
		},
	}
	registerGlobalFlags(root.PersistentFlags(), g)

	root.AddCommand(
		newTUICmd(g),
		newVersionCmd(g),
		newInitCmd(g),
		newCreateCmd(g),
		newShowCmd(g),
		newEditCmd(g),
		newMoveCmd(g),
		newAssignCmd(g),
		newLinkCmd(g),
		newCommentCmd(g),
		newListCmd(g),
		newTreeCmd(g),
		newFindCmd(g),
		newDiscoverCmd(g),
		newHooksCmd(g),
		newAutomationCmd(g),
		newWorkonCmd(g),
		newNowCmd(g),
		newSyncCmd(g),
		newCommitCmd(g),
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
		newLabelCmd(g),
		newConfigCmd(g),
		newChangesCmd(g),
		newSchemaCmd(g),
	)
	attachCompletions(root, g)
	return root, g
}

func chdirArgFrom(args []string) string {
	g := &globalFlags{}
	fs := pflag.NewFlagSet("chdirArg", pflag.ContinueOnError)
	fs.ParseErrorsWhitelist.UnknownFlags = true
	fs.Usage = func() {}
	fs.SetOutput(io.Discard)
	registerGlobalFlags(fs, g)
	_ = fs.Parse(args)
	return g.chdir
}

// DisableFlagParsing hands a subcommand its full argv, global flags and all,
// with only cmdName itself removed. This splits that argv back into the
// global-flag prefix and the subcommand's own args, locating cmdName in argv
// (the real os.Args) to find the boundary — which is why a bridged
// (SetArgs-based) Execute() call, where cmdName never appears in os.Args,
// passes rest through unsplit.
func stripGlobalPrefix(argv, args []string, cmdName string) (chdir string, rest []string) {
	i := slices.Index(argv, cmdName)
	if i >= 0 && i <= len(args) {
		return chdirArgFrom(args[:i]), args[i:]
	}
	return "", args
}

func openStore(g *globalFlags) (*core.Store, *datamodel.Config, error) {
	s, err := core.Discover(g.chdir, g.prompter())
	if err != nil {
		return nil, nil, err
	}
	cfg, err := s.Config()
	if err != nil {
		return nil, nil, err
	}
	return s, cfg, nil
}
