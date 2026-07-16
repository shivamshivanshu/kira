package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func newConfigCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect project config and manage user preferences",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, ok := config.UserConfigDir(os.Getenv)
			if !ok {
				return errx.Env("cannot resolve user config directory: set HOME or XDG_CONFIG_HOME")
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), dir)
			renderPresentFiles(cmd.OutOrStdout(), config.PresentUserFiles(dir))
			return nil
		},
	}
	cmd.AddCommand(newConfigInitCmd(g), newConfigSetCmd(g), newConfigFiltersCmd(g))
	return cmd
}

func newConfigInitCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Scaffold the user config directory with commented defaults",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, err := config.InitUser(os.Getenv)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderConfigInit(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func renderConfigInit(w io.Writer, res *datamodel.ConfigInitResult) {
	if res.Created {
		_, _ = fmt.Fprintf(w, "Created user config in %s\n", res.Path)
		for _, f := range res.Files {
			_, _ = fmt.Fprintf(w, "  %s\n", f)
		}
		return
	}
	_, _ = fmt.Fprintf(w, "%s already exists; leaving it untouched\n", res.Path)
	renderPresentFiles(w, res.Files)
}

func renderPresentFiles(w io.Writer, files []string) {
	if len(files) > 0 {
		_, _ = fmt.Fprintf(w, "  present: %s\n", strings.Join(files, ", "))
	} else {
		_, _ = fmt.Fprintln(w, "  present: none")
	}
}

func newConfigFiltersCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "filters",
		Short: "List the named saved queries from config filters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res := core.Filters(cfg)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderFilterList(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func renderFilterList(w io.Writer, res *datamodel.FilterListResult) {
	if len(res.Filters) == 0 {
		_, _ = fmt.Fprintln(w, "no filters configured")
		return
	}
	tw := newTabWriter(w)
	for _, f := range res.Filters {
		_, _ = fmt.Fprintf(tw, "%s\t%s\n", f.Name, f.Query)
	}
	_ = tw.Flush()
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
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", res.Key, res.Value)
			return nil
		},
	}
}
