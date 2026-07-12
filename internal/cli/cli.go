// Package cli wires the kira cobra command tree. cmd/kira is a thin main that
// delegates here; every command is a thin argv/flag adapter over internal/core,
// which holds the one implementation of each verb (docs/design/01-architecture.md §6).
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/core"
)

// Main runs the root command and returns a process exit code, mapping a
// core.Error to its exit code and any other failure to 1 (docs/design/04-cli.md §1).
// Errors print on stderr, keeping stdout free for --json consumers.
func Main() int {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "kira:", err)
		var ce *core.Error
		if errors.As(err, &ce) {
			return ce.Code
		}
		return 1
	}
	return 0
}

// globalFlags are the persistent flags shared by every subcommand
// (docs/design/04-cli.md §2). noColor and quiet are accepted for forward
// compatibility but unused in M0: human output carries no ANSI color yet, and
// there are no suppressible nags until the index/staleness layer lands.
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
	}
	pf := root.PersistentFlags()
	pf.BoolVar(&g.json, "json", false, "emit machine-readable JSON on stdout")
	pf.BoolVar(&g.noColor, "no-color", false, "disable ANSI color in human output")
	pf.StringVarP(&g.chdir, "C", "C", "", "run as if invoked from `path`")
	pf.BoolVar(&g.quiet, "quiet", false, "suppress non-essential human output")

	root.AddCommand(
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
		newTreeCmd(g),
		newFindCmd(g),
		newDiscoverCmd(g),
		newHooksCmd(g),
	)
	return root
}

// openStore discovers the .kira/ store relative to the -C path (or cwd) and
// loads its config, the common preamble of every command except init.
func openStore(g *globalFlags) (*core.Store, *config.Config, error) {
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
