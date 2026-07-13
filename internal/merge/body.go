package merge

import (
	"sort"
	"time"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func mergeBody(base, ours, theirs string, winner Side, tm TextMerger) string {
	baseProse, baseComments := codec.SplitComments(base)
	oursProse, oursComments := codec.SplitComments(ours)
	theirsProse, theirsComments := codec.SplitComments(theirs)

	prose := mergeProse(baseProse, oursProse, theirsProse, winner, tm)
	comments := unionComments(baseComments, oursComments, theirsComments)
	return codec.JoinComments(prose, comments)
}

func mergeProse(base, ours, theirs string, winner Side, tm TextMerger) string {
	switch {
	case ours == theirs:
		return ours
	case ours == base:
		return theirs
	case theirs == base:
		return ours
	}
	if merged, conflict := tm(base, ours, theirs); !conflict {
		return merged
	}
	if winner == Ours {
		return ours
	}
	return theirs
}

func unionComments(groups ...[]datamodel.Comment) []datamodel.Comment {
	byID := map[string]datamodel.Comment{}
	for _, g := range groups {
		for _, c := range g {
			if _, ok := byID[c.ID]; !ok {
				byID[c.ID] = c
			}
		}
	}
	out := make([]datamodel.Comment, 0, len(byID))
	for _, c := range byID {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		ti, ei := time.Parse(time.RFC3339, out[i].Ts)
		tj, ej := time.Parse(time.RFC3339, out[j].Ts)
		if ei == nil && ej == nil && !ti.Equal(tj) {
			return ti.Before(tj)
		}
		return out[i].ID < out[j].ID
	})
	return out
}
