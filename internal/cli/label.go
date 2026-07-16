package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func newLabelCmd(g *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label",
		Short: "Manage the label vocabulary and item labels",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newLabelCreateCmd(g), newLabelListCmd(g), newLabelAddCmd(g), newLabelRmCmd(g))
	return cmd
}

func newLabelCreateCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>...",
		Short: "Register one or more labels in the config vocabulary",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.LabelCreate(cfg, args)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if g.json {
				return emitJSON(out, res)
			}
			for _, n := range res.Created {
				_, _ = fmt.Fprintf(out, "Created label %s\n", n)
			}
			for _, n := range res.AlreadyKnown {
				_, _ = fmt.Fprintf(out, "Label %s already exists\n", n)
			}
			return nil
		},
	}
}

func newLabelListCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List labels with per-label item counts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.LabelList(cfg)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			renderLabelList(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func renderLabelList(w io.Writer, res *datamodel.LabelListResult) {
	if len(res.Labels) == 0 {
		_, _ = fmt.Fprintln(w, "no labels")
		return
	}
	tw := newTabWriter(w)
	_, _ = fmt.Fprintln(tw, "LABEL\tITEMS")
	for _, l := range res.Labels {
		_, _ = fmt.Fprintf(tw, "%s\t%d\n", l.Name, l.Count)
	}
	_ = tw.Flush()
}

func newLabelAddCmd(g *globalFlags) *cobra.Command { return newLabelMutateCmd(g, true) }
func newLabelRmCmd(g *globalFlags) *cobra.Command  { return newLabelMutateCmd(g, false) }

func newLabelMutateCmd(g *globalFlags, add bool) *cobra.Command {
	use, short := "add <id>... <label>", "Add a label to one or more items without replacing the label set"
	if !add {
		use, short = "rm <id>... <label>", "Remove a label from one or more items without replacing the label set"
	}
	var force bool
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids, label := args[:len(args)-1], args[len(args)-1]
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			b, err := s.BeginBatch(cfg)
			if err != nil {
				return err
			}
			defer b.Close()
			if b.RefExists(label) {
				return errx.User("%q resolves to an existing item, not a label", label).WithHint("the last argument must be the label; ids come first")
			}
			apply := func(id string) (*datamodel.MutationResult, error) {
				return b.LabelSet(id, label, add, force)
			}
			line := func(res *datamodel.MutationResult) string { return labelLine(res, label, add) }
			warn := func(w io.Writer, res *datamodel.MutationResult) { emitMutationWarnings(w, res.Warnings) }
			out := cmd.OutOrStdout()
			return runSingleOrBulk(out, cmd.ErrOrStderr(), g.json, ids, apply, line, warn)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "accept a label outside the configured vocabulary")
	return cmd
}

func labelLine(res *datamodel.MutationResult, label string, add bool) string {
	if len(res.Changed) == 0 {
		if add {
			return fmt.Sprintf("%s: already labelled %s", res.Number, label)
		}
		return fmt.Sprintf("%s: not labelled %s", res.Number, label)
	}
	if add {
		return fmt.Sprintf("Added %s to %s", label, res.Number)
	}
	return fmt.Sprintf("Removed %s from %s", label, res.Number)
}
