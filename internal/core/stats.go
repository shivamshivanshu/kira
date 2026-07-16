package core

import (
	"slices"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

type StatsOpts struct {
	Epic   string
	Since  string
	Weeks  int
	Sprint string
}

func (s *Store) Stats(cfg *datamodel.Config, opts StatsOpts) (*datamodel.StatsResult, error) {
	ld, err := s.read(cfg, loadOpts{useIndex: true})
	if err != nil {
		return nil, err
	}
	items, resolver := ld.items, ld.resolver
	now := time.Now().Local()

	scope, set, err := s.resolveScope(cfg, opts, items, resolver)
	if err != nil {
		return nil, err
	}

	heads := s.fileHeads()
	mis := make([]metricItem, 0, len(set))
	for _, it := range set {
		mi, err := s.itemMetrics(cfg, it, heads[it.ID])
		if err != nil {
			return nil, err
		}
		mis = append(mis, mi)
	}

	return &datamodel.StatsResult{
		Scope:      scope,
		Completion: computeCompletion(mis),
		CycleTime:  computeCycle(mis),
		LeadTime:   computeLead(mis),
		Throughput: computeThroughput(mis, scope.Weeks, now),
		Reopens:    computeReopens(mis),
	}, nil
}

func (s *Store) resolveScope(cfg *datamodel.Config, opts StatsOpts, items []*datamodel.Item, resolver *id.Resolver) (*datamodel.StatsScope, []*datamodel.Item, error) {
	scope := &datamodel.StatsScope{Weeks: opts.Weeks, Since: opts.Since}
	if scope.Weeks <= 0 {
		scope.Weeks = defaultWeeks
	}

	var since time.Time
	if opts.Since != "" {
		var err error
		since, err = time.ParseInLocation(time.DateOnly, opts.Since, time.Local)
		if err != nil {
			return nil, nil, errx.User("--since %q: %v", opts.Since, err)
		}
	}

	set := items
	if opts.Epic != "" {
		ulid, err := resolveID(resolver, opts.Epic)
		if err != nil {
			return nil, nil, err
		}
		epic := findByULID(items, ulid)
		if epic == nil {
			return nil, nil, errx.User("resolved %s to %s, which has no file", opts.Epic, ulid)
		}
		scope.Epic, scope.EpicNumber = ulid, epic.Number
		set, err = descendants(items, ulid)
		if err != nil {
			return nil, nil, err
		}
	}

	var sprintKey string
	if opts.Sprint != "" {
		var err error
		sprintKey, err = s.ResolveSprintKey(cfg, opts.Sprint)
		if err != nil {
			return nil, nil, err
		}
		scope.Sprint = sprintKey
	}

	if sprintKey != "" || opts.Since != "" {
		set = slices.DeleteFunc(slices.Clone(set), func(it *datamodel.Item) bool {
			if sprintKey != "" && !inSprint(it, sprintKey) {
				return true
			}
			if opts.Since == "" {
				return false
			}
			created, err := it.CreatedTime()
			return err != nil || created.Before(since)
		})
	}
	return scope, set, nil
}

func descendants(items []*datamodel.Item, epicULID string) ([]*datamodel.Item, error) {
	var out []*datamodel.Item
	err := walkEpic(epicChildren(items), epicULID, func(*datamodel.Item) bool { return true }, func(c *datamodel.Item) {
		out = append(out, c)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
