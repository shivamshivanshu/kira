package core

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

const fmFence = "---"

func walkPatch(stream string, emit func(path string, events []datamodel.Event)) {
	var d fmDiffState
	var sha, ts, path string
	minus, plus := map[string]string{}, map[string]string{}
	flush := func() {
		emit(path, fmEvents(d.created, minus, plus, ts, sha))
		d.reset()
		minus, plus = map[string]string{}, map[string]string{}
	}
	for _, line := range strings.Split(stream, "\n") {
		switch {
		case strings.HasPrefix(line, "\x00"):
			flush()
			path = ""
			if before, after, ok := strings.Cut(line[1:], "\x00"); ok {
				sha, ts = before, after
			}
		case strings.HasPrefix(line, "diff --git "):
			flush()
			path = ""
		case strings.HasPrefix(line, "+++ b/"):
			path = line[len("+++ b/"):]
		default:
			if added, k, v, ok := d.step(line); ok {
				if added {
					plus[k] = v
				} else {
					minus[k] = v
				}
			}
		}
	}
	flush()
}

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
