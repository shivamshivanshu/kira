package cli

import (
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

// version is the build version, overridable at link time:
//
//	go build -ldflags "-X github.com/shivamshivanshu/kira/internal/cli.version=v1.2.3"
var version = "dev"

func newVersionCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the kira version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if g.json {
				return emitJSON(cmd.OutOrStdout(), versionResult())
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
			return err
		},
	}
}

func versionResult() datamodel.VersionResult {
	res := datamodel.VersionResult{
		Version:      version,
		JSONContract: datamodel.JSONContractVersion,
		Go:           runtime.Version(),
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				res.Commit = s.Value
				break
			}
		}
	}
	return res
}
