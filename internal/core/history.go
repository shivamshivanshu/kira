package core

import (
	"slices"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

type transitionEvent struct {
	ts   time.Time
	from string
	to   string
}

func (s *Store) stateEvents(ulid string) ([]transitionEvent, error) {
	rel := s.fs().RelToRoot(s.itemPath(ulid))
	out, err := s.repo().FollowLogPatch(rel)
	if err != nil {
		if strings.Contains(err.Error(), "does not have any commits") {
			return nil, nil
		}
		return nil, errx.User("%s", err)
	}

	var newestFirst []transitionEvent
	var ts time.Time
	var from, to string
	flushPendingCommit := func() {
		if to != "" {
			newestFirst = append(newestFirst, transitionEvent{ts: ts, from: from, to: to})
		}
		from, to = "", ""
	}
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "\x01"):
			flushPendingCommit()
			t, perr := time.Parse(time.RFC3339, strings.TrimSpace(line[1:]))
			if perr != nil {
				return nil, errx.User("parsing commit date %q: %v", line[1:], perr)
			}
			ts = t
		case strings.HasPrefix(line, "-state: "):
			from = unquoteScalar(line[len("-state: "):])
		case strings.HasPrefix(line, "+state: "):
			to = unquoteScalar(line[len("+state: "):])
		}
	}
	flushPendingCommit()
	slices.Reverse(newestFirst)
	return newestFirst, nil
}

func unquoteScalar(v string) string {
	return strings.Trim(strings.TrimSpace(v), `"'`)
}

type doneInfo struct {
	doneDay  string
	degraded bool
}

func (s *Store) itemDoneInfo(cfg *datamodel.Config, it *datamodel.Item) (doneInfo, error) {
	evs, err := s.stateEvents(it.ID)
	if err != nil {
		return doneInfo{}, err
	}
	wf, hasWorkflow := cfg.Workflows[it.Type]
	var di doneInfo
	for _, ev := range evs {
		if di.doneDay == "" && isDoneState(cfg, it.Type, ev.to) {
			di.doneDay = ev.ts.Local().Format(time.DateOnly)
		}
		offGraphJump := ev.from != "" && hasWorkflow && wf.EnforceTransitions && !transitionAllowed(wf, ev.from, ev.to)
		if offGraphJump {
			di.degraded = true
		}
	}
	if di.doneDay == "" && isDoneState(cfg, it.Type, it.State) {
		if updated, terr := it.UpdatedTime(); terr == nil {
			di.doneDay = updated.Local().Format(time.DateOnly)
			di.degraded = true
		}
	}
	return di, nil
}
