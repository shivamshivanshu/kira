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
		newHooksCmd(g),
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
		newSchemaCmd(),
	)
	attachCompletions(root, g)
	return root, g
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
