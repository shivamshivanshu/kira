package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func newEditCmd(g *globalFlags) *cobra.Command {
	var (
		fields   []string
		labels   []string
		fromFile string
		force    bool
		toEpic   bool
	)
	typedFlags := []struct {
		name, usage string
		value       *string
	}{
		{"title", "set title", new(string)},
		{"subtype", "set subtype (empty clears)", new(string)},
		{"priority", "set priority (empty clears)", new(string)},
		{"rank", "set rank (empty clears)", new(string)},
		{"owner", "set owner (empty clears)", new(string)},
		{"sprint", "set sprint key (empty clears)", new(string)},
		{"due", "set due date (RFC3339 date; empty clears)", new(string)},
		{"resolution", "set resolution directly (out-of-band correction; normally set by move)", new(string)},
	}
	promotionConflicts := []string{"field", "label", "from-file", "force"}
	cmd := &cobra.Command{
		Use:   "edit <id>...",
		Short: "Edit one or more tickets or epics",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if toEpic {
				return runPromotion(cmd, g, args, promotionConflicts)
			}
			opts := core.EditOpts{FromFile: fromFile, Force: force}
			edits, err := parseFieldEdits(fields, "priority=P1")
			if err != nil {
				return err
			}
			opts.Fields = edits
			for _, tf := range typedFlags {
				if cmd.Flags().Changed(tf.name) {
					opts.Fields = append(opts.Fields, core.FieldEdit{Key: tf.name, Value: *tf.value})
				}
			}
			if cmd.Flags().Changed("label") {
				opts.Fields = append(opts.Fields, core.FieldEdit{Key: datamodel.KeyLabels, Value: strings.Join(labels, ",")})
			}
			if err := g.rejectStdinSource(fromFile); err != nil {
				return err
			}
			if g.nonInteractive && fromFile == "" && len(opts.Fields) == 0 {
				return errx.User("edit without field flags opens $EDITOR").WithHint("the editor is unavailable in non-interactive mode; use --field key=value or --from-file <path>")
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			apply := func(id string) (*datamodel.MutationResult, error) { return s.Edit(cfg, id, opts) }
			out := cmd.OutOrStdout()
			if len(args) != 1 {
				if fromFile != "" {
					return errx.User("--from-file cannot be combined with multiple ids")
				}
				if len(opts.Fields) == 0 {
					return errx.User("editing multiple ids requires field flags; editor mode is single-id only")
				}
				batch, err := s.BeginBatch(cfg)
				if err != nil {
					return err
				}
				defer batch.Close()
				apply = func(id string) (*datamodel.MutationResult, error) {
					return batch.EditFields(id, opts.Fields, opts.Force)
				}
			}
			return runSingleOrBulk(out, cmd.ErrOrStderr(), g.json, args, apply, editLine, warnMutation)
		},
	}
	f := cmd.Flags()
	f.StringArrayVar(&fields, "field", nil, "key=value edit (repeatable); skips $EDITOR")
	f.StringArrayVar(&labels, "label", nil, "set labels (repeatable; replaces the label set)")
	for _, tf := range typedFlags {
		f.StringVar(tf.value, tf.name, "", tf.usage)
		promotionConflicts = append(promotionConflicts, tf.name)
	}
	f.StringVar(&fromFile, "from-file", "", "round-trip an edited item file (or - for stdin)")
	f.BoolVar(&force, "force", false, "accept field values outside the configured vocabulary and bypass workflow guards on state changes")
	f.BoolVar(&toEpic, "to-epic", false, "promote a single ticket to an epic")
	return cmd
}

func runPromotion(cmd *cobra.Command, g *globalFlags, args, conflictingFlags []string) error {
	if len(args) != 1 {
		return errx.User("--to-epic requires exactly one ticket id")
	}
	for _, name := range conflictingFlags {
		if cmd.Flags().Changed(name) {
			return errx.User("--to-epic cannot be combined with --%s", name)
		}
	}
	s, cfg, err := openStore(g)
	if err != nil {
		return err
	}
	apply := func(ref string) (*datamodel.MutationResult, error) { return s.PromoteTicket(cfg, ref) }
	return runSingleOrBulk(cmd.OutOrStdout(), cmd.ErrOrStderr(), g.json, args, apply, editLine, warnMutation)
}

func editLine(res *datamodel.MutationResult) string {
	if len(res.Changed) == 0 {
		return fmt.Sprintf("%s: no changes", res.Number)
	}
	return fmt.Sprintf("Edited %s: %s", res.Number, strings.Join(res.Changed, ", "))
}
