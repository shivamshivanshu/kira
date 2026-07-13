package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
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
			for _, arg := range args {
				if isProjectConfig(s.Root(), arg) {
					return errx.User("%s is the project config, not a ticket file; use kira doctor to check it", arg)
				}
				if fi, err := os.Stat(arg); err == nil && fi.Mode().IsRegular() {
					data, err := os.ReadFile(arg)
					if err != nil {
						return errx.User("reading %s: %v", arg, err)
					}
					targets = append(targets, doctor.File{Path: arg, Content: string(data)})
					continue
				}
				path, content, err := s.ResolveItemFile(cfg, arg)
				if err != nil {
					return errx.User("%q is neither a file nor a resolvable ticket id", arg)
				}
				targets = append(targets, doctor.File{Path: path, Content: content})
			}
			report := doctor.Validate(cfg, storeFiles, targets)
			if err := renderReport(cmd.OutOrStdout(), report, g.json); err != nil {
				return err
			}
			return reportExit(report, "validate")
		},
	}
}

func isProjectConfig(root, arg string) bool {
	cfgPath, cfgErr := filepath.Abs(filepath.Join(root, storage.ConfigRelPath))
	argPath, argErr := filepath.Abs(arg)
	return cfgErr == nil && argErr == nil && cfgPath == argPath
}
