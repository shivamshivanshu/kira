package cli

import (
	"github.com/spf13/cobra"
)

func newValidateCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "validate <file>...",
		Short: "Validate ticket files against the schema and config",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			report, err := s.ValidateFiles(cfg, args)
			if err != nil {
				return err
			}
			if err := renderReport(cmd.OutOrStdout(), report, g.json); err != nil {
				return err
			}
			return reportExit(report, "validate")
		},
	}
}
