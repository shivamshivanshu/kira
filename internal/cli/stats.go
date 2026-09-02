package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/statsfmt"
)

func newStatsCmd(g *globalFlags) *cobra.Command {
	var opts core.StatsOpts
	cmd := &cobra.Command{
		Use:   "stats [parent-id] [--since DATE] [--weeks N] [--sprint KEY]",
		Short: "Project and sprint metrics: completion, cycle/lead time, throughput",
		Args:  cobra.MaximumNArgs(1),
		RunE: storeActionRunE(g,
			func(s *core.Store, cfg *datamodel.Config, args []string) (*datamodel.StatsResult, error) {
				if len(args) == 1 {
					opts.Parent = args[0]
				}
				return s.Stats(cfg, opts)
			},
			renderStats),
	}
	f := cmd.Flags()
	f.StringVar(&opts.Since, "since", "", "include only items created on or after this date (YYYY-MM-DD, by each item's recorded calendar day)")
	f.IntVar(&opts.Weeks, "weeks", 0, "trailing weeks in the throughput series (default 8)")
	f.StringVar(&opts.Sprint, "sprint", "", "scope metrics to a sprint ('active' resolves the local active sprint)")
	return cmd
}

func renderStats(out io.Writer, res *datamodel.StatsResult) {
	if sc := res.Scope; sc != nil {
		var parts []string
		if sc.ParentNumber != "" {
			parts = append(parts, "parent "+sc.ParentNumber)
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
		_, _ = fmt.Fprintf(out, "Stats (%s)\n", scope)
	}
	if c := res.Completion; c != nil {
		_, _ = fmt.Fprintf(out, "  completion: %s\n", statsfmt.CompletionLine(c))
	}
	renderPercentiles(out, "cycle time", res.CycleTime)
	renderPercentiles(out, "lead time", res.LeadTime)
	if len(res.Throughput) > 0 {
		nums := make([]string, len(res.Throughput))
		for i, n := range res.Throughput {
			nums[i] = strconv.Itoa(n)
		}
		_, _ = fmt.Fprintf(out, "  throughput/week: %s\n", strings.Join(nums, " "))
	}
	if r := res.Reopens; r != nil && r.Count > 0 {
		_, _ = fmt.Fprintf(out, "  reopens: %d across %s\n", r.Count, strings.Join(r.Items, ", "))
	}
}

func renderPercentiles(out io.Writer, label string, p *datamodel.Percentiles) {
	if p == nil || p.N == 0 {
		return
	}
	_, _ = fmt.Fprintf(out, "  %s (days): %s", label, statsfmt.PercentileLine(p))
	if p.DegradedN > 0 {
		_, _ = fmt.Fprintf(out, "  (%d best-effort)", p.DegradedN)
	}
	_, _ = fmt.Fprintln(out)
}
