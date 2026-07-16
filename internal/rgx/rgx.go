package rgx

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	ScannerInitialBuffer = 64 * 1024
	ScannerMaxLineSize   = 4 * 1024 * 1024
)

type Line struct {
	Path    string
	IsMatch bool
	LineNo  int
	Text    string
}

func Installed() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

var lineRE = regexp.MustCompile(`^(.*?\.md)([:-])(\d+)[:-](.*)$`)

func ParseLine(s string) (Line, bool) {
	m := lineRE.FindStringSubmatch(s)
	if m == nil {
		return Line{}, false
	}
	n, err := strconv.Atoi(m[3])
	if err != nil {
		return Line{}, false
	}
	return Line{Path: m[1], IsMatch: m[2] == ":", LineNo: n, Text: m[4]}, true
}

// Search owns the full rg invocation's flag ordering: passthru first, then
// the enforced output flags, so enforced flags win over a passthru attempt
// to override them (e.g. --heading), and only then the "--" path terminator.
func Search(dir string, passthru []string, path string) ([]Line, error) {
	rgArgs := make([]string, 0, len(passthru)+6)
	rgArgs = append(rgArgs, passthru...)
	rgArgs = append(rgArgs, "--line-number", "--no-heading", "--with-filename", "--color=never", "--", path)

	cmd := exec.Command("rg", rgArgs...)
	cmd.Dir = dir
	out, err := cmd.Output()

	var runErr error
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return nil, fmt.Errorf("running ripgrep: %v", err)
		}
		if ee.ExitCode() == 1 {
			return nil, nil
		}
		msg := strings.TrimSpace(string(ee.Stderr))
		if msg == "" {
			msg = "ripgrep failed"
		}
		runErr = errors.New(msg)
	}

	lines, scanErr := scanLines(out)
	return lines, errors.Join(runErr, scanErr)
}

func scanLines(out []byte) ([]Line, error) {
	var lines []Line
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 0, ScannerInitialBuffer), ScannerMaxLineSize)
	for scanner.Scan() {
		raw := scanner.Text()
		l, ok := ParseLine(raw)
		if !ok {
			l = Line{Text: raw}
		}
		lines = append(lines, l)
	}
	return lines, scanner.Err()
}
