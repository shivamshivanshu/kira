package merge

import (
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

// mergeExtras merges the schema-agnostic extra frontmatter fields, keeping
// ExtraScalars and ExtraLists disjoint: each key lands in exactly one of the
// two returned maps, never both (a duplicate key across both would make the
// serialized frontmatter unparseable).
func mergeExtras(base, ours, theirs *datamodel.Item, winner Side) (map[string]string, map[string][]string) {
	keys := keyUnion(base.ExtraScalars, ours.ExtraScalars, theirs.ExtraScalars)
	for k := range keyUnion(base.ExtraLists, ours.ExtraLists, theirs.ExtraLists) {
		keys[k] = true
	}

	scalars := map[string]string{}
	lists := map[string][]string{}
	for k := range keys {
		oursScalar, oursIsScalar := ours.ExtraScalars[k]
		oursList, oursIsList := ours.ExtraLists[k]
		theirsScalar, theirsIsScalar := theirs.ExtraScalars[k]
		theirsList, theirsIsList := theirs.ExtraLists[k]

		switch {
		case (oursIsScalar && theirsIsList) || (oursIsList && theirsIsScalar):
			if winner == Ours {
				if oursIsList {
					lists[k] = oursList
				} else {
					scalars[k] = oursScalar
				}
			} else {
				if theirsIsList {
					lists[k] = theirsList
				} else {
					scalars[k] = theirsScalar
				}
			}
		case oursIsList || theirsIsList:
			if merged := setMerge(base.ExtraLists[k], ours.ExtraLists[k], theirs.ExtraLists[k]); len(merged) > 0 {
				lists[k] = merged
			}
		default:
			if merged := threeWayPtr(mapEntry(base.ExtraScalars, k), mapEntry(ours.ExtraScalars, k), mapEntry(theirs.ExtraScalars, k), winner); merged != nil {
				scalars[k] = *merged
			}
		}
	}

	if len(scalars) == 0 {
		scalars = nil
	}
	if len(lists) == 0 {
		lists = nil
	}
	return scalars, lists
}

// changedExtraKeys reports which extra scalar/list keys differ between a and
// b, mirroring datamodel.ChangedFields for the schema-agnostic extras maps so
// arbitrated() can detect an extras-only three-way conflict.
func changedExtraKeys(a, b *datamodel.Item) []string {
	var out []string
	for k := range extraKeyUnion(a, b) {
		if extraChanged(a, b, k) {
			out = append(out, k)
		}
	}
	return out
}

func extraKeyUnion(a, b *datamodel.Item) map[string]bool {
	keys := keyUnion(a.ExtraScalars, b.ExtraScalars)
	for k := range keyUnion(a.ExtraLists, b.ExtraLists) {
		keys[k] = true
	}
	return keys
}

func extraChanged(a, b *datamodel.Item, key string) bool {
	as, aIsScalar := a.ExtraScalars[key]
	bs, bIsScalar := b.ExtraScalars[key]
	if aIsScalar || bIsScalar {
		return aIsScalar != bIsScalar || as != bs
	}
	return !slices.Equal(a.ExtraLists[key], b.ExtraLists[key])
}
