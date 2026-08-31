package merge

import (
	"slices"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func mergeBody(base, ours, theirs string, winner Side, tm TextMerger) string {
	baseProse, baseComments, baseCanonical := codec.SplitComments(base)
	oursProse, oursComments, oursCanonical := codec.SplitComments(ours)
	theirsProse, theirsComments, theirsCanonical := codec.SplitComments(theirs)
	if !baseCanonical || !oursCanonical || !theirsCanonical {
		return mergeProse(base, ours, theirs, winner, tm)
	}

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
	type datedComment struct {
		comment datamodel.Comment
		ts      time.Time
		valid   bool
	}
	ordered := make([]datedComment, 0, len(byID))
	for _, c := range byID {
		ts, err := time.Parse(time.RFC3339, c.Ts)
		ordered = append(ordered, datedComment{c, ts, err == nil})
	}
	slices.SortFunc(ordered, func(a, b datedComment) int {
		if a.valid && b.valid && !a.ts.Equal(b.ts) {
			return a.ts.Compare(b.ts)
		}
		return strings.Compare(a.comment.ID, b.comment.ID)
	})
	out := make([]datamodel.Comment, len(ordered))
	for i, c := range ordered {
		out[i] = c.comment
	}
	return out
}
