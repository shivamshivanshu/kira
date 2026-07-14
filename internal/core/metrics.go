package core

import (
	"math"
	"slices"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

const defaultWeeks = 8

func computeCompletion(items []metricItem) *datamodel.Completion {
	c := &datamodel.Completion{Total: len(items)}
	for _, it := range items {
		switch {
		case it.dropped:
			c.Dropped++
		case it.category == datamodel.CategoryDone:
			c.Done++
		}
	}
	if c.Total > 0 {
		c.Pct = round2(float64(c.Done) / float64(c.Total))
	}
	return c
}

func computeCycle(items []metricItem) *datamodel.Percentiles {
	var vals []float64
	degraded := 0
	for _, it := range items {
		if it.dropped || !it.hasDoing || !it.hasDone {
			continue
		}
		d := it.doneAt.Sub(it.doingAt).Hours() / 24
		if d < 0 {
			continue
		}
		vals = append(vals, d)
		if it.degraded {
			degraded++
		}
	}
	return percentiles(vals, degraded)
}

func computeLead(items []metricItem) *datamodel.Percentiles {
	var vals []float64
	for _, it := range items {
		if it.dropped || !it.hasDone || it.created.IsZero() {
			continue
		}
		d := it.doneAt.Sub(it.created).Hours() / 24
		if d < 0 {
			d = 0
		}
		vals = append(vals, d)
	}
	return percentiles(vals, 0)
}

func computeThroughput(items []metricItem, weeks int, today time.Time) []int {
	buckets := make([]int, weeks)
	for _, it := range items {
		if !it.hasDone || it.dropped {
			continue
		}
		daysAgo := int(today.Sub(it.doneAt).Hours() / 24)
		if daysAgo < 0 {
			daysAgo = 0
		}
		w := daysAgo / 7
		if w >= weeks {
			continue
		}
		buckets[weeks-1-w]++
	}
	return buckets
}

func computeReopens(items []metricItem) *datamodel.Reopens {
	r := &datamodel.Reopens{Items: []string{}}
	for _, it := range items {
		if it.reopens > 0 {
			r.Count += it.reopens
			r.Items = append(r.Items, it.number)
		}
	}
	return r
}

func percentiles(vals []float64, degraded int) *datamodel.Percentiles {
	p := &datamodel.Percentiles{N: len(vals), DegradedN: degraded}
	if len(vals) == 0 {
		return p
	}
	slices.Sort(vals)
	p.P50 = round1(percentile(vals, 0.5))
	p.P90 = round1(percentile(vals, 0.9))
	return p
}

func percentile(sorted []float64, q float64) float64 {
	rank := q * float64(len(sorted)-1)
	lo := int(math.Floor(rank))
	if lo+1 >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	return sorted[lo] + (rank-float64(lo))*(sorted[lo+1]-sorted[lo])
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }

func round1(x float64) float64 { return math.Round(x*10) / 10 }
