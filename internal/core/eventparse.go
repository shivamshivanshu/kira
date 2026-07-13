package core

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
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

var scalarFields = scalarFieldSet()

func scalarFieldSet() map[string]bool {
	nonScalar := map[string]bool{
		datamodel.KeyAliases: true, datamodel.KeyLabels: true,
		datamodel.KeyBlockedBy: true, datamodel.KeyLinks: true,
		datamodel.KeyCreated: true, datamodel.KeyUpdated: true,
	}
	scalar := map[string]bool{}
	for _, k := range datamodel.FrontmatterKeys {
		if !nonScalar[k] {
			scalar[k] = true
		}
	}
	return scalar
}

func frontmatterField(line string) (key, value string, ok bool) {
	colon := strings.IndexByte(line, ':')
	if colon <= 0 {
		return "", "", false
	}
	key = line[:colon]
	if !scalarFields[key] {
		return "", "", false
	}
	return key, unquoteScalar(line[colon+1:]), true
}

func unquoteScalar(v string) string {
	return strings.Trim(strings.TrimSpace(v), `"'`)
}

func fmEvents(created bool, minus, plus map[string]string, ts, sha string) []datamodel.Event {
	if created {
		return nil
	}
	var events []datamodel.Event
	for _, field := range datamodel.FrontmatterKeys {
		mv, hadMinus := minus[field]
		pv, hadPlus := plus[field]
		if (hadMinus || hadPlus) && mv != pv {
			events = append(events, datamodel.Event{Ts: ts, Field: field, Old: mv, New: pv, CommitSHA: sha})
		}
	}
	return events
}

func (s *Store) deriveEvents(ulid string) ([]datamodel.Event, error) {
	out, err := s.repo().FileLog(s.fs().RelToRoot(s.itemPath(ulid)))
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits") {
			return nil, nil
		}
		return nil, errx.User("%s", err)
	}
	var events []datamodel.Event
	walkPatch(out, func(_ string, evs []datamodel.Event) {
		events = append(events, evs...)
	})
	return events, nil
}

func (s *Store) rangeEvents(repo gitx.Repo, sinceSHA, headSHA string, tracked map[string]*datamodel.Item) (map[string][]datamodel.Event, error) {
	fs := s.fs()
	out, err := repo.RangePatch(sinceSHA+".."+headSHA, fs.RelToRoot(fs.ItemsDir()))
	if err != nil {
		return nil, errx.User("%s", err)
	}
	events := map[string][]datamodel.Event{}
	walkPatch(out, func(path string, evs []datamodel.Event) {
		if len(evs) == 0 {
			return
		}
		if ulid := storage.ULIDFromPath(path); tracked[ulid] != nil {
			events[ulid] = append(events[ulid], evs...)
		}
	})
	return events, nil
}

func (s *Store) cachedEvents(ulid, fileHead string) (events []datamodel.Event, committed bool, err error) {
	events, _, err = index.LogEntries(s.fs(), ulid, fileHead, func() ([]datamodel.Event, error) {
		return s.deriveEvents(ulid)
	})
	return events, fileHead != "", err
}
