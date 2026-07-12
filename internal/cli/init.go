package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

func newInitCmd(g *globalFlags) *cobra.Command {
	var key string
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a .kira/ store in the current git repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := core.Init(g.chdir, key, force)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Initialized kira in %s (project key %s)\n", res.Path, res.ProjectKey)
			return nil
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "project key for display IDs (default: derived from directory name)")
	cmd.Flags().BoolVar(&force, "force", false, "reinitialize over an existing .kira/")
	return cmd
}
