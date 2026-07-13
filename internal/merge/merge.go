// Package merge implements kira's field-level three-way auto-merge policy over
// parsed items, shared by the git merge driver and the in-process sync path.
package merge

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

type Result struct {
	Item       *datamodel.Item
	Arbitrated []string
}

func Merge(base, ours, theirs *datamodel.Item, remote Side, tm TextMerger) Result {
	if base == nil {
		base = &datamodel.Item{}
	}
	winner := laterUpdated(ours.Updated, theirs.Updated, remote)

	m := &datamodel.Item{
		ID:         ours.ID,
		Number:     ours.Number,
		Type:       ours.Type,
		Created:    ours.Created,
		Aliases:    aliasUnion(base.Aliases, ours.Aliases, theirs.Aliases),
		Subtype:    threeWayPtr(base.Subtype, ours.Subtype, theirs.Subtype, winner),
		Title:      threeWayScalar(base.Title, ours.Title, theirs.Title, winner),
		State:      threeWayScalar(base.State, ours.State, theirs.State, winner),
		Resolution: threeWayPtr(base.Resolution, ours.Resolution, theirs.Resolution, winner),
		Priority:   threeWayPtr(base.Priority, ours.Priority, theirs.Priority, winner),
		Rank:       threeWayPtr(base.Rank, ours.Rank, theirs.Rank, winner),
		Owner:      threeWayPtr(base.Owner, ours.Owner, theirs.Owner, winner),
		Reporter:   threeWayPtr(base.Reporter, ours.Reporter, theirs.Reporter, winner),
		Labels:     setMerge(base.Labels, ours.Labels, theirs.Labels),
		Epic:       threeWayPtr(base.Epic, ours.Epic, theirs.Epic, winner),
		BlockedBy:  setMerge(base.BlockedBy, ours.BlockedBy, theirs.BlockedBy),
		Links:      linkMerge(base.Links, ours.Links, theirs.Links),
		Sprint:     threeWayPtr(base.Sprint, ours.Sprint, theirs.Sprint, winner),
		Due:        threeWayPtr(base.Due, ours.Due, theirs.Due, winner),
		Estimate:   threeWayPtr(base.Estimate, ours.Estimate, theirs.Estimate, winner),
		Updated:    maxUpdated(ours.Updated, theirs.Updated),
		Body:       mergeBody(base.Body, ours.Body, theirs.Body, winner, tm),
	}
	return Result{Item: m, Arbitrated: arbitrated(base, ours, theirs)}
}

func arbitrated(base, ours, theirs *datamodel.Item) []string {
	fromBaseOurs := asSet(datamodel.ChangedFields(base, ours))
	fromBaseTheirs := asSet(datamodel.ChangedFields(base, theirs))
	var out []string
	for _, f := range datamodel.ChangedFields(ours, theirs) {
		if fromBaseOurs[f] && fromBaseTheirs[f] {
			out = append(out, f)
		}
	}
	return out
}

func maxUpdated(ours, theirs string) string {
	if laterUpdated(ours, theirs, Ours) == Ours {
		return ours
	}
	return theirs
}
