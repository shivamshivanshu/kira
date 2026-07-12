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
	n, _ := strconv.Atoi(m[3])
	return Line{Path: m[1], IsMatch: m[2] == ":", LineNo: n, Text: m[4]}, true
}

func Search(dir string, args []string) ([]Line, bool, error) {
	rgArgs := append([]string{
		"--line-number", "--no-heading", "--with-filename", "--color=never",
	}, args...)
	cmd := exec.Command("rg", rgArgs...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ee.ExitCode() == 1 {
				return nil, false, nil
			}
			msg := strings.TrimSpace(string(ee.Stderr))
			if msg == "" {
				msg = "ripgrep failed"
			}
			return nil, false, errors.New(msg)
		}
		return nil, false, fmt.Errorf("running ripgrep: %v", err)
	}

	var lines []Line
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		raw := scanner.Text()
		l, ok := ParseLine(raw)
		if !ok {
			l = Line{Text: raw}
		}
		lines = append(lines, l)
	}
	return lines, true, nil
}
