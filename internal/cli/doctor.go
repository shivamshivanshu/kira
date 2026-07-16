package cli

import (
	"cmp"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func newDoctorCmd(g *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check the repo for id collisions, dangling refs, and schema violations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, cfg, err := openStore(g)
			if err != nil {
				return err
			}
			report, err := s.DoctorReport(cfg)
			if err != nil {
				return err
			}
			if err := renderReport(cmd.OutOrStdout(), report, g.json); err != nil {
				return err
			}
			return reportExit(report, "doctor")
		},
	}
}

func renderReport(w io.Writer, report *doctor.Report, asJSON bool) error {
	if asJSON {
		return emitJSON(w, report)
	}
	for _, f := range report.Findings {
		_, _ = fmt.Fprintf(w, "%-7s %-15s %s%s\n", f.Severity, f.Class, findingLocation(f), f.Message)
	}
	_, _ = fmt.Fprintf(w, "%s: %d %s, %d %s, %d info\n",
		reportVerdict(report),
		report.Summary.Error, plural(report.Summary.Error, "error"),
		report.Summary.Warning, plural(report.Summary.Warning, "warning"),
		report.Summary.Info)
	return nil
}

func findingLocation(f doctor.Finding) string {
	loc := cmp.Or(f.Number, f.ItemID, f.Path)
	if loc == "" {
		return ""
	}
	if f.Field != "" {
		return loc + " " + f.Field + ": "
	}
	return loc + ": "
}

func reportVerdict(report *doctor.Report) string {
	if report.OK {
		return "ok"
	}
	return "problems found"
}

func reportExit(report *doctor.Report, cmd string) error {
	if report.OK {
		return nil
	}
	return errx.User("%s: %d %s found", cmd, report.Summary.Error, plural(report.Summary.Error, "problem"))
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}
