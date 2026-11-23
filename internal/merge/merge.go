// Package merge implements kira's field-level three-way auto-merge policy over
// parsed items, shared by the git merge driver and the in-process sync path.
package merge

import (
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/ptr"
	"github.com/shivamshivanshu/kira/internal/setx"
)

type Result struct {
	Item             *datamodel.Item
	Arbitrated       []string
	WorkflowConflict bool
}

func Merge(base, ours, theirs *datamodel.Item, remote Side, tm TextMerger) Result {
	if ours.Type != theirs.Type && !isUncontestedPromotion(base, ours, theirs) {
		return Result{WorkflowConflict: true}
	}
	if base == nil {
		base = &datamodel.Item{}
	}
	winner := laterUpdated(ours.Updated, theirs.Updated, remote)
	number := threeWayScalar(base.Number, ours.Number, theirs.Number, winner)
	aliases := slices.DeleteFunc(aliasUnion(base.Aliases, ours.Aliases, theirs.Aliases), func(a string) bool { return a == number })

	m := &datamodel.Item{
		ID:         ours.ID,
		Number:     number,
		Type:       threeWayScalar(base.Type, ours.Type, theirs.Type, winner),
		Created:    ours.Created,
		Aliases:    aliases,
		Subtype:    threeWayPtr(base.Subtype, ours.Subtype, theirs.Subtype, winner),
		Title:      threeWayScalar(base.Title, ours.Title, theirs.Title, winner),
		State:      threeWayScalar(base.State, ours.State, theirs.State, winner),
		Resolution: threeWayPtr(base.Resolution, ours.Resolution, theirs.Resolution, winner),
		Priority:   threeWayPtr(base.Priority, ours.Priority, theirs.Priority, winner),
		Rank:       threeWayPtr(base.Rank, ours.Rank, theirs.Rank, winner),
		Owner:      threeWayPtr(base.Owner, ours.Owner, theirs.Owner, winner),
		Reporter:   threeWayPtr(base.Reporter, ours.Reporter, theirs.Reporter, winner),
		Labels:     setMerge(base.Labels, ours.Labels, theirs.Labels),
		Parent:     threeWayPtr(base.Parent, ours.Parent, theirs.Parent, winner),
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

func isUncontestedPromotion(base, ours, theirs *datamodel.Item) bool {
	ticket, epic := ours, theirs
	if ours.Type == datamodel.TypeEpic {
		ticket, epic = theirs, ours
	}
	if base == nil || base.Type != datamodel.TypeTicket ||
		ticket.Type != datamodel.TypeTicket || epic.Type != datamodel.TypeEpic {
		return false
	}
	return ticket.State == base.State && ptr.Equal(ticket.Subtype, base.Subtype) && ptr.Equal(ticket.Resolution, base.Resolution)
}

func arbitrated(base, ours, theirs *datamodel.Item) []string {
	fromBaseOurs := setx.ToSet(datamodel.ChangedFields(base, ours))
	fromBaseTheirs := setx.ToSet(datamodel.ChangedFields(base, theirs))

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
