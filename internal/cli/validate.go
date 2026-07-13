package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/errx"
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
			storeFiles, err := readTicketFiles(s)
			if err != nil {
				return err
			}
			targets := make([]doctor.File, 0, len(args))
			for _, path := range args {
				data, err := os.ReadFile(path)
				if err != nil {
					return errx.User("reading %s: %v", path, err)
				}
				targets = append(targets, doctor.File{Path: path, Content: string(data)})
			}
			report := doctor.Validate(cfg, storeFiles, targets)
			if err := renderReport(cmd.OutOrStdout(), report, g.json); err != nil {
				return err
			}
			return reportExit(report, "validate")
		},
	}
}
