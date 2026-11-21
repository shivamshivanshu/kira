package core

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/index"
	"github.com/shivamshivanshu/kira/internal/setx"
	"github.com/shivamshivanshu/kira/internal/storage"
)

const fmFence = "---"

// eventFields is the frontmatter key set the diff walker reports on, in
// report order: every key except the created/updated timestamps.
type eventFields struct {
	tracked map[string]bool
	order   []string
}

var trackedEventFields = newEventFields()

func newEventFields() eventFields {
	untracked := map[string]bool{datamodel.KeyCreated: true, datamodel.KeyUpdated: true}
	f := eventFields{tracked: map[string]bool{}}
	for _, key := range datamodel.FrontmatterKeys {
		if !untracked[key] {
			f.tracked[key] = true
			f.order = append(f.order, key)
		}
	}
	return f
}

func walkPatch(stream string, emit func(oldPath, newPath string, events []datamodel.Event)) {
	var d fmDiffState
	var sha, ts, oldPath, newPath string
	inHunk := false
	flush := func() {
		emit(oldPath, newPath, d.events(ts, sha))
		d = fmDiffState{
			before: frontmatterLines{lines: d.before.lines[:0]},
			after:  frontmatterLines{lines: d.after.lines[:0]},
		}
		oldPath, newPath, inHunk = "", "", false
	}
	for line := range strings.SplitSeq(stream, "\n") {
		switch {
		case strings.HasPrefix(line, "\x00"):
			flush()
			if before, after, ok := strings.Cut(line[1:], "\x00"); ok {
				sha, ts = before, after
			}
		case strings.HasPrefix(line, "diff --git "):
			flush()
		case !inHunk && strings.HasPrefix(line, "--- a/"):
			oldPath = line[len("--- a/"):]
		case !inHunk && strings.HasPrefix(line, "+++ b/"):
			newPath = line[len("+++ b/"):]
		case strings.HasPrefix(line, "@@ "):
			inHunk = true
		default:
			d.step(line)
		}
	}
	flush()
}

// fmDiffState rebuilds the before and after frontmatter of one file from its
// patch. That only works because gitx.logPatch asks git for whole-file
// context; under a narrower -U a hunk can open mid-frontmatter and every field
// change in it is dropped silently.
type fmDiffState struct {
	created bool
	before  frontmatterLines
	after   frontmatterLines
}

type frontmatterLines struct {
	fences int
	lines  []string
}

func (f *frontmatterLines) add(content string) {
	if content == fmFence {
		f.fences++
		return
	}
	if f.fences == 1 {
		f.lines = append(f.lines, content)
	}
}

func (d *fmDiffState) step(line string) {
	if strings.HasPrefix(line, "--- /dev/null") {
		d.created = true
		return
	}
	if line == "" || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
		return
	}
	op, content := line[0], line[1:]
	if op == ' ' || op == '-' {
		d.before.add(content)
	}
	if op == ' ' || op == '+' {
		d.after.add(content)
	}
}

func (d *fmDiffState) events(ts, sha string) []datamodel.Event {
	if d.created || slices.Equal(d.before.lines, d.after.lines) {
		return nil
	}
	before, after := frontmatterValues(d.before.lines), frontmatterValues(d.after.lines)
	var events []datamodel.Event
	for _, field := range trackedEventFields.order {
		if before[field] != after[field] {
			events = append(events, datamodel.Event{Ts: ts, Field: field, Old: before[field], New: after[field], CommitSHA: sha})
		}
	}
	return events
}

// frontmatterValues renders each tracked field of a frontmatter block as a
// comparable string. An empty list reads as absent, so a format migration that
// only adds `labels: []` lines records no event. Links span a nested block of
// `  <type>: [...]` lines and render like their `kira diff` form.
func frontmatterValues(lines []string) map[string]string {
	values := map[string]string{}
	var links []string
	inLinks := false
	for _, line := range lines {
		if inLinks && strings.HasPrefix(line, " ") {
			if typ, targets, ok := strings.Cut(strings.TrimSpace(line), ":"); ok {
				if v := nonEmptyList(codec.DecodeScalar(targets)); v != "" {
					links = append(links, typ+":"+v)
				}
			}
			continue
		}
		key, raw, ok := strings.Cut(line, ":")
		inLinks = ok && key == datamodel.KeyLinks
		if !ok || !trackedEventFields.tracked[key] {
			continue
		}
		if v := nonEmptyList(codec.DecodeScalar(raw)); v != "" {
			values[key] = v
		}
	}
	if len(links) > 0 {
		slices.Sort(links)
		values[datamodel.KeyLinks] = strings.Join(links, " ")
	}
	return values
}

func nonEmptyList(value string) string {
	if value == "[]" {
		return ""
	}
	return value
}

func (s *Store) deriveEvents(ulid string) ([]datamodel.Event, error) {
	return s.deriveEventsAt("", ulid)
}

func (s *Store) deriveEventsAt(revision, ulid string) ([]datamodel.Event, error) {
	out, err := s.repo().FileLog(revision, s.fs().RelToRoot(s.itemPath(ulid)))
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits") {
			return nil, nil
		}
		return nil, errx.User("%s", err)
	}
	var events []datamodel.Event
	walkPatch(out, func(_, _ string, evs []datamodel.Event) {
		events = append(events, evs...)
	})
	return events, nil
}

func (s *Store) deriveEventBatch(revision string, ids []string) (map[string][]datamodel.Event, error) {
	out := make(map[string][]datamodel.Event, len(ids))
	if len(ids) > 1 && revision != "" {
		s.deriveFirstParentEvents(revision, ids, out)
	}
	for _, id := range ids {
		if _, batched := out[id]; batched {
			continue
		}
		events, err := s.deriveEventsAt(revision, id)
		if err != nil {
			return nil, err
		}
		out[id] = events
	}
	return out, nil
}

func (s *Store) deriveFirstParentEvents(revision string, ids []string, out map[string][]datamodel.Event) {
	repo := s.repo()
	if follows, err := repo.LogFollowsRenames(); follows || err != nil {
		return
	}
	itemsDir := filepath.ToSlash(s.fs().RelToRoot(s.fs().ItemsDir()))
	mergePaths, err := repo.FirstParentMergePaths(revision, itemsDir)
	if err != nil {
		return
	}
	// A single-path log can leave the first-parent chain only at a merge
	// whose tree entry differs from the first parent's.
	changesAtMerge := setx.ToSet(mergePaths)
	pathToID := make(map[string]string, len(ids))
	paths := make([]string, 0, len(ids))
	for _, id := range ids {
		path := filepath.ToSlash(s.fs().RelToRoot(s.itemPath(id)))
		if !changesAtMerge[path] {
			pathToID[path] = id
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return
	}
	if len(paths) > 64 {
		paths = []string{itemsDir}
	}
	stream, err := repo.FirstParentFileLogs(revision, paths)
	if err != nil {
		return
	}
	for _, id := range pathToID {
		out[id] = nil
	}
	walkPatch(stream, func(oldPath, newPath string, events []datamodel.Event) {
		if newPath == "" {
			newPath = oldPath
		}
		if id, wanted := pathToID[newPath]; wanted {
			out[id] = append(out[id], events...)
		}
	})
}

func (s *Store) rangeEvents(repo gitx.Repo, sinceSHA, headSHA string, tracked map[string]*datamodel.Item) (map[string][]datamodel.Event, error) {
	fs := s.fs()
	out, err := repo.RangePatch(sinceSHA+".."+headSHA, fs.RelToRoot(fs.ItemsDir()))
	if err != nil {
		return nil, errx.User("%s", err)
	}
	events := map[string][]datamodel.Event{}
	walkPatch(out, func(_, path string, evs []datamodel.Event) {
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
