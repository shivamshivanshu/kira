package core

import "strings"

const fmFence = "---"

type fmDiffState struct {
	fences  int
	created bool
}

func (d *fmDiffState) reset() {
	d.fences = 0
	d.created = false
}

func (d *fmDiffState) step(line string) (added bool, key, value string, isField bool) {
	if strings.HasPrefix(line, "--- /dev/null") {
		d.created = true
		return
	}
	if line == "" || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") ||
		strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "diff ") ||
		strings.HasPrefix(line, "index ") {
		return
	}
	op, content := line[0], line[1:]
	if content == fmFence {
		if op == ' ' || op == '+' {
			d.fences++
		}
		return
	}
	if op != '+' && op != '-' {
		return
	}
	if d.fences != 1 {
		return
	}
	k, v, ok := frontmatterField(content)
	if !ok {
		return
	}
	return op == '+', k, v, true
}
