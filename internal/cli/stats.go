package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
)

func newStatsCmd(g *globalFlags) *cobra.Command {
	var opts core.StatsOpts
	cmd := &cobra.Command{
		Use:   "stats [--sprint KEY] [--velocity]",
		Short: "Sprint burndown and velocity metrics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			res, err := s.Stats(cfg, opts)
			if err != nil {
				return err
			}
			if g.json {
				return emitJSON(cmd.OutOrStdout(), res)
			}
			out := cmd.OutOrStdout()
			if b := res.Burndown; b != nil {
				fmt.Fprintf(out, "Burndown %s  %s -> %s (%s)\n", b.Sprint, b.Start, b.End, b.Unit)
				for _, d := range b.Days {
					fmt.Fprintf(out, "  %s  remaining %s  ideal %s\n", d.Date, codec.EmitFloat(d.Remaining), codec.EmitFloat(d.Ideal))
				}
				if b.Unestimated > 0 {
					fmt.Fprintf(out, "  unestimated items (burn nothing): %d\n", b.Unestimated)
				}
				if b.DegradedN > 0 {
					fmt.Fprintf(out, "  items with lossy history (best-effort done day): %d\n", b.DegradedN)
				}
			}
			if v := res.Velocity; v != nil {
				fmt.Fprintf(out, "Velocity (%s)\n", v.Unit)
				for _, sp := range v.Sprints {
					fmt.Fprintf(out, "  %s  %s\n", sp.Key, codec.EmitFloat(sp.Completed))
				}
				fmt.Fprintf(out, "  trailing-3 average: %s\n", codec.EmitFloat(v.Trailing3))
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.Sprint, "sprint", "", "sprint key to report burndown for ('active' resolves the local active sprint)")
	f.BoolVar(&opts.Velocity, "velocity", false, "report completed estimate per closed sprint and the trailing-3 average")
	return cmd
}
