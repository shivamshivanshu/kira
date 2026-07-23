package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func printHookLines(w io.Writer, hooks []datamodel.AutomationHookView) {
	for _, h := range hooks {
		name := h.Name
		if name == "" {
			name = "(unnamed)"
		}
		line := fmt.Sprintf("hook %s on %s: %s", name, h.On, h.Run)
		if h.Source == datamodel.AutomationSourceUser {
			line += " (user)"
		}
		if !h.Enabled {
			line += " [disabled]"
		}
		_, _ = fmt.Fprintln(w, line)
	}
}

func newAutomationCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "automation",
		Short: "Manage kira event automation hooks",
	}
	cmd.AddCommand(
		newAutomationListCmd(g),
		newAutomationTrustCmd(g),
	)
	return cmd
}

func newAutomationListCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List defined automation hooks and local trust status",
		Args:  cobra.NoArgs,
		RunE: storeActionRunE(g,
			func(s *core.Store, cfg *datamodel.Config, _ []string) (*datamodel.AutomationListResult, error) {
				return s.AutomationList(cfg), nil
			},
			printAutomationList),
	}
}

func newAutomationTrustCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "trust",
		Short: "Trust the current automation config so its hooks may run locally",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if !g.json {
				printHookLines(out, s.AutomationList(cfg).Hooks)
			}
			res, err := s.AutomationTrust(cfg)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(out, res)
			}
			_, _ = fmt.Fprintf(out, "trusted %d automation hooks\n", len(res.Hooks))
			return nil
		},
	}
}

func printAutomationList(out io.Writer, res *datamodel.AutomationListResult) {
	if len(res.Hooks) == 0 {
		_, _ = fmt.Fprintln(out, "no automation hooks defined")
		return
	}
	printHookLines(out, res.Hooks)
	if res.Trusted {
		_, _ = fmt.Fprintln(out, "trust: trusted")
	} else {
		_, _ = fmt.Fprintln(out, "trust: not trusted (run `kira automation trust`)")
	}
}
