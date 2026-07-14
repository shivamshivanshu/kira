package core

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

type commitMeta struct {
	author  string
	ts      string
	parents int
}

func (s *Store) Blame(cfg *datamodel.Config, ref string) (*datamodel.BlameResult, error) {
	ld, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		return nil, err
	}
	items, resolver := ld.items, ld.resolver
	ulid, err := resolveID(resolver, ref)
	if err != nil {
		return nil, err
	}
	it := findByULID(items, ulid)
	if it == nil {
		return nil, errx.User("resolved %s to %s, which has no file", ref, ulid)
	}

	events, _, err := s.cachedEvents(ulid, s.fileHead(ulid))
	if err != nil {
		return nil, err
	}
	meta, newest, creation, err := s.fileCommitMeta(ulid)
	if err != nil {
		return nil, err
	}

	latest := map[string]datamodel.Event{}
	for _, e := range events {
		if _, seen := latest[e.Field]; !seen {
			latest[e.Field] = e
		}
	}
	current := scalarFieldValues(it)

	res := &datamodel.BlameResult{ID: ulid, Number: it.Number, Fields: []datamodel.BlameField{}}
	for _, field := range datamodel.FrontmatterKeys {
		if !scalarFields[field] {
			continue
		}
		val, hasVal := current[field]
		ev, hasEv := latest[field]
		switch {
		case hasEv:
			cm := meta[ev.CommitSHA]
			bf := datamodel.BlameField{Field: field, Value: val, When: ev.Ts, By: cm.author, SourceKind: datamodel.BlameSourceCommit, Degraded: cm.parents > 1}
			if hasVal && val != ev.New {
				bf.SourceKind, bf.Degraded, bf.When, bf.By = datamodel.BlameSourceSynthetic, true, newest.ts, newest.author
			}
			res.Fields = append(res.Fields, bf)
		case hasVal:
			if val == "null" {
				continue
			}
			res.Fields = append(res.Fields, datamodel.BlameField{Field: field, Value: val, When: creation.ts, By: creation.author, SourceKind: datamodel.BlameSourceCreated})
		}
	}
	return res, nil
}

func (s *Store) fileCommitMeta(ulid string) (bySHA map[string]commitMeta, newest, creation commitMeta, err error) {
	out, err := s.repo().FileCommitMeta(s.fs().RelToRoot(s.itemPath(ulid)))
	if err != nil {
		return nil, commitMeta{}, commitMeta{}, errx.User("%s", err)
	}
	bySHA = map[string]commitMeta{}
	newestSeen := false
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\x00")
		if len(f) != 4 {
			continue
		}
		cm := commitMeta{author: f[1], ts: f[2], parents: len(strings.Fields(f[3]))}
		bySHA[f[0]] = cm
		if !newestSeen {
			newestSeen = true
			newest = cm
		}
		creation = cm
	}
	return bySHA, newest, creation, nil
}

func scalarFieldValues(it *datamodel.Item) map[string]string {
	front := strings.TrimPrefix(codec.Serialize(it), codec.FenceLine)
	if i := strings.Index(front, codec.FenceLine); i >= 0 {
		front = front[:i]
	}
	v := map[string]string{}
	for _, line := range strings.Split(front, "\n") {
		if k, val, ok := frontmatterField(line); ok {
			v[k] = val
		}
	}
	return v
}
