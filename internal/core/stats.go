package core

import (
	"math"
	"slices"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

type StatsOpts struct {
	Epic     string
	Since    string
	Weeks    int
	Sprint   string
	Velocity bool
}

func (s *Store) Stats(cfg *datamodel.Config, opts StatsOpts) (*datamodel.StatsResult, error) {
	items, _, resolver, _, err := s.indexedLoad(cfg)
	if err != nil {
		return nil, err
	}
	now := time.Now().Local()
	today := now.Format(time.DateOnly)
	unit := string(cfg.Estimate.Unit)

	memo := map[string]metricItem{}
	metricsOf := func(it *datamodel.Item) (metricItem, error) {
		if mi, ok := memo[it.ID]; ok {
			return mi, nil
		}
		mi, err := s.itemMetrics(cfg, it)
		if err != nil {
			return metricItem{}, err
		}
		memo[it.ID] = mi
		return mi, nil
	}

	scope, set, sprintKey, err := s.resolveScope(cfg, opts, items, resolver)
	if err != nil {
		return nil, err
	}

	mis := make([]metricItem, 0, len(set))
	for _, it := range set {
		mi, err := metricsOf(it)
		if err != nil {
			return nil, err
		}
		mis = append(mis, mi)
	}

	res := &datamodel.StatsResult{
		Scope:      scope,
		Completion: computeCompletion(mis),
		CycleTime:  computeCycle(mis),
		LeadTime:   computeLead(mis),
		Throughput: computeThroughput(mis, scope.Weeks, now),
		Estimate:   computeEstimate(mis, unit, cfg.Estimate.HoursPerDay),
		Reopens:    computeReopens(mis),
	}

	if opts.Sprint != "" {
		sp, _ := cfg.Sprint(sprintKey)
		res.Burndown = computeBurndown(sp, unit, mis, today)
	}
	if opts.Velocity {
		closed := closedSprints(cfg, today)
		bySprint := map[string][]metricItem{}
		for _, it := range items {
			if it.Sprint == nil || !slices.ContainsFunc(closed, func(sp datamodel.Sprint) bool { return sp.Key == *it.Sprint }) {
				continue
			}
			mi, err := metricsOf(it)
			if err != nil {
				return nil, err
			}
			bySprint[*it.Sprint] = append(bySprint[*it.Sprint], mi)
		}
		res.Velocity = computeVelocity(closed, unit, bySprint)
	}
	return res, nil
}

func (s *Store) resolveScope(cfg *datamodel.Config, opts StatsOpts, items []*datamodel.Item, resolver *id.Resolver) (*datamodel.StatsScope, []*datamodel.Item, string, error) {
	scope := &datamodel.StatsScope{Weeks: opts.Weeks, Since: opts.Since}
	if scope.Weeks <= 0 {
		scope.Weeks = defaultWeeks
	}

	set := items
	if opts.Epic != "" {
		ulid, err := resolveID(resolver, opts.Epic)
		if err != nil {
			return nil, nil, "", err
		}
		epic := findByULID(items, ulid)
		if epic == nil {
			return nil, nil, "", errx.User("resolved %s to %s, which has no file", opts.Epic, ulid)
		}
		scope.Epic, scope.EpicNumber = ulid, epic.Number
		set, err = descendants(items, ulid)
		if err != nil {
			return nil, nil, "", err
		}
	}

	var sprintKey string
	if opts.Sprint != "" {
		var err error
		sprintKey, err = s.ResolveSprintKey(cfg, opts.Sprint)
		if err != nil {
			return nil, nil, "", err
		}
		scope.Sprint = sprintKey
	}

	if sprintKey != "" || opts.Since != "" {
		set = slices.DeleteFunc(slices.Clone(set), func(it *datamodel.Item) bool {
			return (sprintKey != "" && !inSprint(it, sprintKey)) || (opts.Since != "" && it.Created < opts.Since)
		})
	}
	return scope, set, sprintKey, nil
}

func descendants(items []*datamodel.Item, epicULID string) ([]*datamodel.Item, error) {
	children := epicChildren(items)
	var out []*datamodel.Item
	onPath := map[string]bool{epicULID: true}
	var walk func(parentID string) error
	walk = func(parentID string) error {
		for _, c := range children[parentID] {
			if onPath[c.ID] {
				return errx.Conflict("epic cycle detected at %s", c.Number)
			}
			onPath[c.ID] = true
			out = append(out, c)
			if err := walk(c.ID); err != nil {
				return err
			}
			delete(onPath, c.ID)
		}
		return nil
	}
	if err := walk(epicULID); err != nil {
		return nil, err
	}
	return out, nil
}

func computeBurndown(sp datamodel.Sprint, unit string, items []metricItem, today string) *datamodel.Burndown {
	b := &datamodel.Burndown{Sprint: sp.Key, Start: sp.Start, End: sp.End, Unit: unit, Days: []datamodel.BurndownDay{}}
	for _, it := range items {
		if !it.estimated {
			b.Unestimated++
		}
		if it.degraded {
			b.DegradedN++
		}
	}
	remainingAt := func(day string) float64 {
		var sum float64
		for _, it := range items {
			if it.doneDay == "" || it.doneDay > day {
				sum += it.estimate
			}
		}
		return sum
	}
	start, err1 := time.Parse(time.DateOnly, sp.Start)
	end, err2 := time.Parse(time.DateOnly, sp.End)
	if err1 != nil || err2 != nil {
		return b
	}
	span := int(end.Sub(start).Hours()/24) + 1
	initialRemaining := remainingAt(sp.Start)
	for i := 0; i < span; i++ {
		day := start.AddDate(0, 0, i).Format(time.DateOnly)
		if day > today {
			break
		}
		b.Days = append(b.Days, datamodel.BurndownDay{
			Date:      day,
			Remaining: remainingAt(day),
			Ideal:     linearIdeal(initialRemaining, i, span),
		})
	}
	return b
}

func linearIdeal(initialRemaining float64, dayIndex, span int) float64 {
	return round1(initialRemaining * float64(span-1-dayIndex) / float64(span-1))
}

func computeVelocity(closed []datamodel.Sprint, unit string, bySprint map[string][]metricItem) *datamodel.Velocity {
	v := &datamodel.Velocity{Unit: unit, Sprints: make([]datamodel.VelocitySprint, 0, len(closed))}
	for _, sp := range closed {
		var completed float64
		for _, it := range bySprint[sp.Key] {
			if it.dropped || it.doneDay == "" || it.doneDay < sp.Start || it.doneDay > sp.End {
				continue
			}
			completed += it.estimate
		}
		v.Sprints = append(v.Sprints, datamodel.VelocitySprint{Key: sp.Key, Completed: completed})
	}
	n := min(3, len(v.Sprints))
	if n > 0 {
		var sum float64
		for _, sp := range v.Sprints[len(v.Sprints)-n:] {
			sum += sp.Completed
		}
		v.Trailing3 = round1(sum / float64(n))
	}
	return v
}

func closedSprints(cfg *datamodel.Config, today string) []datamodel.Sprint {
	var closed []datamodel.Sprint
	for _, sp := range cfg.Sprints {
		if sp.End < today {
			closed = append(closed, sp)
		}
	}
	slices.SortFunc(closed, func(a, b datamodel.Sprint) int { return strings.Compare(a.End, b.End) })
	return closed
}

func round1(x float64) float64 { return math.Round(x*10) / 10 }
