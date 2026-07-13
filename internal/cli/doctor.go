package cli

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/storage"
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
			files, err := readTicketFiles(s)
			if err != nil {
				return err
			}
			report := doctor.Run(cfg, files, gatherEnv(s.Root()))
			if err := renderReport(cmd.OutOrStdout(), report, g.json); err != nil {
				return err
			}
			return reportExit(report, "doctor")
		},
	}
}

func readTicketFiles(s *core.Store) ([]doctor.File, error) {
	dir := storage.New(s.Root()).ItemsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errx.User("reading tickets: %v", err)
	}
	var files []doctor.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			continue
		}
		if _, ok := storage.ULIDFromFilename(name); !ok {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, errx.User("reading %s: %v", name, err)
		}
		files = append(files, doctor.File{Path: name, Content: string(data)})
	}
	return files, nil
}

func renderReport(w io.Writer, report *doctor.Report, asJSON bool) error {
	if asJSON {
		return emitJSON(w, report)
	}
	for _, f := range report.Findings {
		fmt.Fprintf(w, "%-7s %-15s %s%s\n", f.Severity, f.Class, findingLocation(f), f.Message)
	}
	fmt.Fprintf(w, "%s: %d error(s), %d warning(s), %d info\n",
		reportVerdict(report), report.Summary.Error, report.Summary.Warning, report.Summary.Info)
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
	return errx.User("%s: %d problem(s) found", cmd, report.Summary.Error)
}
