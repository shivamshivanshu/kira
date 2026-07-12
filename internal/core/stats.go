package core

import (
	"math"
	"slices"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

type StatsOpts struct {
	Sprint   string
	Velocity bool
}

func (s *Store) Stats(cfg *datamodel.Config, opts StatsOpts) (*datamodel.StatsResult, error) {
	if opts.Sprint == "" && !opts.Velocity {
		return nil, errx.User("general project metrics are not implemented yet (M2); pass --sprint <key> and/or --velocity")
	}
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	today := time.Now().Local().Format(time.DateOnly)
	unit := string(cfg.Estimate.Unit)

	memo := map[string]doneInfo{}
	infoOf := func(it *datamodel.Item) (doneInfo, error) {
		if di, ok := memo[it.ID]; ok {
			return di, nil
		}
		di, err := s.itemDoneInfo(cfg, it)
		if err != nil {
			return doneInfo{}, err
		}
		memo[it.ID] = di
		return di, nil
	}

	res := &datamodel.StatsResult{}
	if opts.Sprint != "" {
		key, err := s.ResolveSprintKey(cfg, opts.Sprint)
		if err != nil {
			return nil, err
		}
		sp, _ := cfg.Sprint(key)
		var bis []burnItem
		for _, it := range items {
			if !inSprint(it, key) {
				continue
			}
			di, err := infoOf(it)
			if err != nil {
				return nil, err
			}
			bis = append(bis, burnItem{
				estimate:  deref(it.Estimate),
				estimated: it.Estimate != nil,
				doneDay:   di.doneDay,
				degraded:  di.degraded,
			})
		}
		res.Burndown = computeBurndown(sp, unit, bis, today)
	}
	if opts.Velocity {
		closed := closedSprints(cfg, today)
		bySprint := map[string][]velocityItem{}
		for _, it := range items {
			if it.Sprint == nil || !slices.ContainsFunc(closed, func(sp datamodel.Sprint) bool { return sp.Key == *it.Sprint }) {
				continue
			}
			di, err := infoOf(it)
			if err != nil {
				return nil, err
			}
			bySprint[*it.Sprint] = append(bySprint[*it.Sprint], velocityItem{
				estimate: deref(it.Estimate),
				doneDay:  di.doneDay,
				dropped:  it.Resolution != nil && *it.Resolution == "dropped",
			})
		}
		res.Velocity = computeVelocity(closed, unit, bySprint)
	}
	return res, nil
}

type burnItem struct {
	estimate  float64
	estimated bool
	doneDay   string
	degraded  bool
}

func computeBurndown(sp datamodel.Sprint, unit string, items []burnItem, today string) *datamodel.Burndown {
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

type velocityItem struct {
	estimate float64
	doneDay  string
	dropped  bool
}

func computeVelocity(closed []datamodel.Sprint, unit string, bySprint map[string][]velocityItem) *datamodel.Velocity {
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

func deref(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func round1(x float64) float64 { return math.Round(x*10) / 10 }
