package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newSprintCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sprint",
		Short: "Manage sprint entities and the active-sprint pointer",
	}
	cmd.AddCommand(
		newSprintCreateCmd(g),
		newSprintListCmd(g),
		newSprintActivateCmd(g),
		newSprintCloseCmd(g),
	)
	return cmd
}

func newSprintCreateCmd(g *globalFlags) *cobra.Command {
	var sp datamodel.Sprint
	cmd := &cobra.Command{
		Use:   "create --key KEY --name NAME --start DATE --end DATE",
		Short: "Append a sprint to config sprints (committed like any config mutation)",
		Args:  cobra.NoArgs,
		RunE: storeActionRunE(g,
			func(s *core.Store, cfg *datamodel.Config, _ []string) (*datamodel.SprintCreateResult, error) {
				return s.SprintCreate(cfg, sp)
			},
			func(w io.Writer, res *datamodel.SprintCreateResult) {
				_, _ = fmt.Fprintf(w, "Created sprint %s (%s -> %s)\n", res.Sprint.Key, res.Sprint.Start, res.Sprint.End)
			}),
	}
	f := cmd.Flags()
	f.StringVar(&sp.Key, "key", "", "sprint key, referenced by item sprint fields")
	f.StringVar(&sp.Name, "name", "", "display name")
	f.StringVar(&sp.Start, "start", "", "start date (YYYY-MM-DD)")
	f.StringVar(&sp.End, "end", "", "end date (YYYY-MM-DD), must be after start")
	return cmd
}

func newSprintListCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured sprints with the active marker and item counts",
		Args:  cobra.NoArgs,
		RunE: storeActionRunE(g,
			func(s *core.Store, cfg *datamodel.Config, _ []string) (*datamodel.SprintListResult, error) {
				return s.SprintList(cfg)
			},
			renderSprintList),
	}
}

func renderSprintList(w io.Writer, res *datamodel.SprintListResult) {
	if len(res.Sprints) == 0 {
		_, _ = fmt.Fprintln(w, "No sprints configured (use `kira sprint create`)")
		return
	}
	for _, row := range res.Sprints {
		marker := " "
		if row.Active {
			marker = "*"
		}
		_, _ = fmt.Fprintf(w, "%s %s  %s  %s -> %s  %d/%d done\n",
			marker, row.Key, row.Name, row.Start, row.End, row.Items.Done, row.Items.Total)
	}
}

func newSprintActivateCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "activate <key>",
		Short: "Set the local active sprint (git-ignored pointer, per clone)",
		Args:  cobra.ExactArgs(1),
		RunE: storeActionRunE(g,
			func(s *core.Store, cfg *datamodel.Config, args []string) (*datamodel.SprintActivateResult, error) {
				return s.SprintActivate(cfg, args[0])
			},
			func(w io.Writer, res *datamodel.SprintActivateResult) {
				if res.Previous != "" && res.Previous != res.Activated {
					_, _ = fmt.Fprintf(w, "Activated sprint %s (was %s)\n", res.Activated, res.Previous)
				} else {
					_, _ = fmt.Fprintf(w, "Activated sprint %s\n", res.Activated)
				}
			}),
	}
}

func newSprintCloseCmd(g *globalFlags) *cobra.Command {
	var moveTo string
	cmd := &cobra.Command{
		Use:   "close <key>",
		Short: "Close a sprint, reporting unfinished items (spillover moves with --move-to)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.SprintClose(cfg, args[0], moveTo)
			if err != nil {
				return err
			}
			emitMutationWarnings(cmd.ErrOrStderr(), res.Warnings)
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Closed sprint %s\n", res.Closed)
			for _, num := range res.Unfinished {
				_, _ = fmt.Fprintf(out, "  unfinished: %s\n", num)
			}
			if res.MovedTo != "" {
				_, _ = fmt.Fprintf(out, "Moved %d unfinished item(s) to %s\n", len(res.Unfinished), res.MovedTo)
			}
			if res.WasActive {
				_, _ = fmt.Fprintln(out, "Cleared the active-sprint pointer")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&moveTo, "move-to", "", "reassign unfinished items to this sprint (one commit per item)")
	return cmd
}
