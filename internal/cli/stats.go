package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func newStatsCmd(g *globalFlags) *cobra.Command {
	var opts core.StatsOpts
	cmd := &cobra.Command{
		Use:   "stats [epic-id] [--since DATE] [--weeks N] [--sprint KEY]",
		Short: "Project and sprint metrics: completion, cycle/lead time, throughput",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Epic = args[0]
			}
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
			renderStats(cmd.OutOrStdout(), res)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.Since, "since", "", "include only items created on or after this date (YYYY-MM-DD)")
	f.IntVar(&opts.Weeks, "weeks", 0, "trailing weeks in the throughput series (default 8)")
	f.StringVar(&opts.Sprint, "sprint", "", "scope metrics to a sprint ('active' resolves the local active sprint)")
	return cmd
}

func renderStats(out io.Writer, res *datamodel.StatsResult) {
	if sc := res.Scope; sc != nil {
		var parts []string
		if sc.EpicNumber != "" {
			parts = append(parts, "epic "+sc.EpicNumber)
		}
		if sc.Sprint != "" {
			parts = append(parts, "sprint "+sc.Sprint)
		}
		if sc.Since != "" {
			parts = append(parts, "since "+sc.Since)
		}
		scope := "project"
		if len(parts) > 0 {
			scope = strings.Join(parts, ", ")
		}
		fmt.Fprintf(out, "Stats (%s)\n", scope)
	}
	if c := res.Completion; c != nil {
		fmt.Fprintf(out, "  completion: %d/%d done (%.0f%%)", c.Done, c.Total, c.Pct*100)
		if c.Dropped > 0 {
			fmt.Fprintf(out, ", %d dropped", c.Dropped)
		}
		fmt.Fprintln(out)
	}
	renderPercentiles(out, "cycle time", res.CycleTime)
	renderPercentiles(out, "lead time", res.LeadTime)
	if len(res.Throughput) > 0 {
		nums := make([]string, len(res.Throughput))
		for i, n := range res.Throughput {
			nums[i] = strconv.Itoa(n)
		}
		fmt.Fprintf(out, "  throughput/week: %s\n", strings.Join(nums, " "))
	}
	if r := res.Reopens; r != nil && r.Count > 0 {
		fmt.Fprintf(out, "  reopens: %d across %s\n", r.Count, strings.Join(r.Items, ", "))
	}
}

func renderPercentiles(out io.Writer, label string, p *datamodel.Percentiles) {
	if p == nil || p.N == 0 {
		return
	}
	fmt.Fprintf(out, "  %s (days): p50 %s  p90 %s  n=%d", label, codec.EmitFloat(p.P50), codec.EmitFloat(p.P90), p.N)
	if p.DegradedN > 0 {
		fmt.Fprintf(out, "  (%d best-effort)", p.DegradedN)
	}
	fmt.Fprintln(out)
}
