package core

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/storage"
)

const fmFence = "---"

// eventFields is the frontmatter key set the diff walker reports on: the
// builtin scalars plus every schema-declared scalar extra, in report order.
// Extras are unioned across schemas rather than scoped to the item's type
// because the walker keys off a file path, whose history spans commits where
// the type or its schema may have differed.
type eventFields struct {
	tracked map[string]bool
	order   []string
}

func newEventFields(schemas map[string]entityschema.Schema) eventFields {
	f := eventFields{tracked: make(map[string]bool, len(scalarFields))}
	for _, key := range datamodel.FrontmatterKeys {
		if scalarFields[key] {
			f.add(key)
		}
	}
	for _, d := range declaredExtras(schemas) {
		if !d.List {
			f.add(d.Name)
		}
	}
	return f
}

func (f *eventFields) add(key string) {
	f.tracked[key] = true
	f.order = append(f.order, key)
}

func (s *Store) eventFields() (eventFields, error) {
	schemas, err := s.Schemas()
	if err != nil {
		return eventFields{}, err
	}
	return newEventFields(schemas), nil
}

func walkPatch(stream string, fields eventFields, emit func(path string, events []datamodel.Event)) {
	var d fmDiffState
	var sha, ts, path string
	minus, plus := map[string]string{}, map[string]string{}
	flush := func() {
		emit(path, fmEvents(d.created, minus, plus, ts, sha, fields))
		d.reset()
		minus, plus = map[string]string{}, map[string]string{}
	}
	for line := range strings.SplitSeq(stream, "\n") {
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
			if added, k, v, ok := d.step(line, fields); ok {
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

// Counting fences to find the frontmatter block only works because
// gitx.logPatch asks git for whole-file context; under a narrower -U a hunk
// can open mid-frontmatter and every field change in it is dropped silently.
type fmDiffState struct {
	fences  int
	created bool
}

func (d *fmDiffState) reset() {
	d.fences = 0
	d.created = false
}

func (d *fmDiffState) step(line string, fields eventFields) (added bool, key, value string, isField bool) {
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
	k, v, ok := frontmatterField(content, fields)
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

func frontmatterField(line string, fields eventFields) (key, value string, ok bool) {
	colon := strings.IndexByte(line, ':')
	if colon <= 0 {
		return "", "", false
	}
	key = line[:colon]
	if !fields.tracked[key] {
		return "", "", false
	}
	return key, codec.DecodeScalar(line[colon+1:]), true
}

func fmEvents(created bool, minus, plus map[string]string, ts, sha string, fields eventFields) []datamodel.Event {
	if created {
		return nil
	}
	var events []datamodel.Event
	for _, field := range fields.order {
		mv, hadMinus := minus[field]
		pv, hadPlus := plus[field]
		if (hadMinus || hadPlus) && mv != pv {
			events = append(events, datamodel.Event{Ts: ts, Field: field, Old: mv, New: pv, CommitSHA: sha})
		}
	}
	return events
}

func (s *Store) deriveEvents(ulid string) ([]datamodel.Event, error) {
	fields, err := s.eventFields()
	if err != nil {
		return nil, err
	}
	out, err := s.repo().FileLog(s.fs().RelToRoot(s.itemPath(ulid)))
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits") {
			return nil, nil
		}
		return nil, errx.User("%s", err)
	}
	var events []datamodel.Event
	walkPatch(out, fields, func(_ string, evs []datamodel.Event) {
		events = append(events, evs...)
	})
	return events, nil
}

// Linear histories have identical per-file and multi-file log simplification.
// At merges Git can choose different parents for different pathspecs; retain
// per-file derivation there so batching never changes historical metrics.
func (s *Store) deriveEventBatch(ids []string) (map[string][]datamodel.Event, error) {
	out := make(map[string][]datamodel.Event, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	if len(ids) == 1 {
		events, err := s.deriveEvents(ids[0])
		out[ids[0]] = events
		return out, err
	}
	merges, err := s.repo().Output("rev-list", "--min-parents=2", "--max-count=1", "HEAD")
	if err != nil || merges != "" {
		for _, id := range ids {
			events, err := s.deriveEvents(id)
			if err != nil {
				return nil, err
			}
			out[id] = events
		}
		return out, nil
	}
	paths := []string{s.fs().RelToRoot(s.fs().ItemsDir())}
	if len(ids) <= 64 {
		paths = make([]string, len(ids))
		for n, id := range ids {
			paths[n] = s.fs().RelToRoot(s.itemPath(id))
		}
	}
	fields, err := s.eventFields()
	if err != nil {
		return nil, err
	}
	stream, err := s.repo().FileLogs(paths)
	if err != nil {
		return nil, errx.User("%s", err)
	}
	for _, id := range ids {
		out[id] = nil
	}
	walkPatch(stream, fields, func(path string, events []datamodel.Event) {
		id := storage.ULIDFromPath(path)
		if _, wanted := out[id]; wanted {
			out[id] = append(out[id], events...)
		}
	})
	return out, nil
}

func (s *Store) rangeEvents(repo gitx.Repo, sinceSHA, headSHA string, tracked map[string]*datamodel.Item) (map[string][]datamodel.Event, error) {
	fs := s.fs()
	fields, err := s.eventFields()
	if err != nil {
		return nil, err
	}
	out, err := repo.RangePatch(sinceSHA+".."+headSHA, fs.RelToRoot(fs.ItemsDir()))
	if err != nil {
		return nil, errx.User("%s", err)
	}
	events := map[string][]datamodel.Event{}
	walkPatch(out, fields, func(path string, evs []datamodel.Event) {
		if len(evs) == 0 {
			return
		}
		if ulid := storage.ULIDFromPath(path); tracked[ulid] != nil {
			events[ulid] = append(events[ulid], evs...)
		}
	})
	return events, nil
}

// cachedEvents returns events newest-first, the order git log emits them in;
// callers that read them chronologically must reverse first.
func (s *Store) cachedEvents(ulid, fileHead string) (events []datamodel.Event, committed bool, err error) {
	events, _, err = index.LogEntries(s.fs(), ulid, fileHead, func() ([]datamodel.Event, error) {
		return s.deriveEvents(ulid)
	})
	return events, fileHead != "", err
}
