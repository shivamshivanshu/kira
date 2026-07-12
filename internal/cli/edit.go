package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
)

func newEditCmd(g *globalFlags) *cobra.Command {
	var (
		fields   []string
		fromFile string
		force    bool
	)
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a ticket or epic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := core.EditOpts{FromFile: fromFile, Force: force}
			for _, f := range fields {
				k, v, ok := strings.Cut(f, "=")
				if !ok {
					return fmt.Errorf("--field %q: expected key=value", f)
				}
				opts.Fields = append(opts.Fields, core.FieldEdit{Key: k, Value: v})
			}
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Edit(cfg, args[0], opts)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			if len(res.Changed) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "%s: no changes\n", res.Number)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Edited %s: %s\n", res.Number, strings.Join(res.Changed, ", "))
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringArrayVar(&fields, "field", nil, "key=value edit (repeatable); skips $EDITOR")
	f.StringVar(&fromFile, "from-file", "", "round-trip an edited item file (or - for stdin)")
	f.BoolVar(&force, "force", false, "bypass strict-vocabulary rejection")
	return cmd
}
