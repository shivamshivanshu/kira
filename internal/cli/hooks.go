package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHooksCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage kira git hooks",
	}
	install := &cobra.Command{
		Use:   "install",
		Short: "Install kira git hooks (not yet implemented)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "kira hooks install: not yet implemented (hook scripts land in M3)")
			return nil
		},
	}
	install.Flags().Bool("with-pre-commit", false, "additionally install the pre-commit hook")
	install.Flags().Bool("validate", false, "verify installed hooks and exit")
	cmd.AddCommand(install)
	return cmd
}
