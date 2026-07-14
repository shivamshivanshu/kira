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
	cmd := &cobra.Command{
		Use:   "edit <id>...",
		Short: "Edit one or more tickets or epics",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := core.EditOpts{FromFile: fromFile, Force: force}
			for _, f := range fields {
				k, v, ok := strings.Cut(f, "=")
				if !ok {
					return fmt.Errorf("--field %q: expected key=value", f)
				}
				opts.Fields = append(opts.Fields, core.FieldEdit{Key: k, Value: v})
			}
			for _, tf := range typedFlags {
				if cmd.Flags().Changed(tf.name) {
					opts.Fields = append(opts.Fields, core.FieldEdit{Key: tf.name, Value: *tf.value})
				}
			}
			if cmd.Flags().Changed("label") {
				opts.Fields = append(opts.Fields, core.FieldEdit{Key: datamodel.KeyLabels, Value: strings.Join(labels, ",")})
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
			}
			return runSingleOrBulk(out, cmd.ErrOrStderr(), g.json, args, apply, editLine)
		},
	}
	f := cmd.Flags()
	f.StringArrayVar(&fields, "field", nil, "key=value edit (repeatable); skips $EDITOR")
	f.StringArrayVar(&labels, "label", nil, "set labels (repeatable; replaces the label set)")
	for _, tf := range typedFlags {
		f.StringVar(tf.value, tf.name, "", tf.usage)
	}
	f.StringVar(&fromFile, "from-file", "", "round-trip an edited item file (or - for stdin)")
	f.BoolVar(&force, "force", false, "accept field values outside the configured vocabulary")
	return cmd
}

func editLine(res *datamodel.MutationResult) string {
	if len(res.Changed) == 0 {
		return fmt.Sprintf("%s: no changes", res.Number)
	}
	return fmt.Sprintf("Edited %s: %s", res.Number, strings.Join(res.Changed, ", "))
}
