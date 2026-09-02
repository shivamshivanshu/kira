// Package cli wires the kira cobra command tree; every command is a thin argv/flag adapter over internal/core.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

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

func Main() int {
	root, g := newRootCmd()
	if err := root.Execute(); err != nil {
		return renderError(os.Stderr, err, g.json)
	}
	return 0
}

func renderError(w io.Writer, err error, jsonMode bool) int {
	code := errx.ExitUser
	hint := ""
	if ce, ok := errors.AsType[*errx.Error](err); ok {
		code, hint = ce.Code, ce.Hint
	}
	crash, isCrash := errors.AsType[*tui.CrashError](err)
	if isCrash {
		code, hint = errx.ExitCrash, crash.LogPath
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
	if isCrash {
		// handleCrash already wrote the human-readable crash report to stderr.
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
	nonInteractive bool
}

func registerGlobalFlags(fs *pflag.FlagSet, g *globalFlags) {
	fs.BoolVar(&g.json, "json", false, "emit machine-readable JSON on stdout")
	fs.BoolVar(&g.noColor, "no-color", false, "disable ANSI color in human output")
	fs.StringVarP(&g.chdir, "C", "C", "", "run as if invoked from `path`")
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
		newServerCmd(g),
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

func globalFlagSet(g *globalFlags) *pflag.FlagSet {
	fs := pflag.NewFlagSet("global", pflag.ContinueOnError)
	fs.ParseErrorsWhitelist.UnknownFlags = true
	fs.Usage = func() {}
	fs.SetOutput(io.Discard)
	registerGlobalFlags(fs, g)
	return fs
}

func chdirArgFrom(args []string) string {
	g := &globalFlags{}
	_ = globalFlagSet(g).Parse(args)
	return g.chdir
}

// globalPrefixLen reports how many leading argv entries the global flags
// consume. Parsing rather than searching for the subcommand name is what stops
// `-C find find pat` from taking the flag's value for the command, and
// non-interspersed parsing stops at the subcommand so its own flags (ripgrep's
// `-C 3`, say) never reach the global set.
func globalPrefixLen(argv []string) int {
	fs := globalFlagSet(&globalFlags{})
	fs.SetInterspersed(false)
	if err := fs.Parse(argv); err != nil {
		return len(argv)
	}
	return len(argv) - len(fs.Args())
}

// DisableFlagParsing hands a subcommand its full argv, global flags and all,
// with only cmdName itself removed. This splits that argv back into the
// global-flag prefix and the subcommand's own args, parsing argv (the real
// os.Args) for where the global flags stop — which is why a bridged
// (SetArgs-based) Execute() call, where cmdName never appears in os.Args,
// passes rest through unsplit.
func stripGlobalPrefix(argv, args []string, cmdName string) (chdir string, rest []string) {
	i := globalPrefixLen(argv)
	if i < len(argv) && argv[i] == cmdName && i <= len(args) {
		return chdirArgFrom(argv[:i]), args[i:]
	}
	return "", args
}

func parseFieldEdits(fields []string, hintExample string) ([]core.FieldEdit, error) {
	var edits []core.FieldEdit
	for _, f := range fields {
		k, v, ok := strings.Cut(f, "=")
		if !ok {
			return nil, errx.User("--field %q: expected key=value", f).WithHint("example: --field %s", hintExample)
		}
		edits = append(edits, core.FieldEdit{Key: k, Value: v})
	}
	return edits, nil
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

func storeActionRunE[T any](g *globalFlags, action func(*core.Store, *datamodel.Config, []string) (T, error), render func(io.Writer, T)) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		s, cfg, err := openStore(g)
		if err != nil {
			return err
		}
		res, err := action(s, cfg, args)
		if err != nil {
			return err
		}
		if g.json {
			return emitJSON(cmd.OutOrStdout(), res)
		}
		render(cmd.OutOrStdout(), res)
		return nil
	}
}
