package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newConfigCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and edit .kira/config.yaml",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newConfigSetCmd(g))
	return cmd
}

func newConfigSetCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a scalar config key, preserving comments and formatting",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.ConfigSet(cfg, args[0], args[1])
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", res.Key, res.Value)
			return nil
		},
	}
}
