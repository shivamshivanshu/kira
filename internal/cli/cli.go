// Package cli wires the kira cobra command tree. cmd/kira is a thin main
// that delegates here; the TUI and other frontends call internal/core, not this.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Main runs the root command and returns a process exit code. It is the single
// entry point shared by cmd/kira and the testscript e2e harness. Errors are
// reported on stderr, keeping stdout free for --json consumers (see 01-architecture §8).
func Main() int {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "kira:", err)
		return 1
	}
	return 0
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "kira",
		Short:         "kira — a git-native, terminal-first ticket tracker",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the kira version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
			return err
		},
	}
}
