package merge

import "github.com/shivamshivanshu/kira/internal/datamodel"

// extraShape is which of an item's three extras maps holds a key. A key lives
// in exactly one of them, so the shape is what decides how it merges.
type extraShape int

const (
	shapeAbsent extraShape = iota
	shapeScalar
	shapeList
	shapeBlock
)

func shapeOf(it *datamodel.Item, k string) extraShape {
	switch {
	case has(it.ExtraScalars, k):
		return shapeScalar
	case has(it.ExtraLists, k):
		return shapeList
	case has(it.ExtraBlocks, k):
		return shapeBlock
	default:
		return shapeAbsent
	}
}

func has[V any](m map[string]V, k string) bool {
	_, ok := m[k]
	return ok
}

// mergeExtras merges the schema-agnostic extra frontmatter fields, keeping the
// three returned maps disjoint: each key lands in exactly one of them, never
// two (a duplicate key across maps would make the serialized frontmatter
// unparseable). A block merges as an opaque unit -- kira does not interpret its
// contents, so it cannot merge inside one.
func mergeExtras(base, ours, theirs *datamodel.Item, winner Side) (map[string]datamodel.Rendered, map[string][]datamodel.Rendered, map[string]datamodel.Rendered) {
	scalars := map[string]datamodel.Rendered{}
	lists := map[string][]datamodel.Rendered{}
	blocks := map[string]datamodel.Rendered{}

	keys := keyUnion(base.ExtraScalars, ours.ExtraScalars, theirs.ExtraScalars)
	for k := range keyUnion(base.ExtraLists, ours.ExtraLists, theirs.ExtraLists) {
		keys[k] = true
	}
	for k := range keyUnion(base.ExtraBlocks, ours.ExtraBlocks, theirs.ExtraBlocks) {
		keys[k] = true
	}

	for k := range keys {
		oursShape, theirsShape := shapeOf(ours, k), shapeOf(theirs, k)
		switch {
		case oursShape != shapeAbsent && theirsShape != shapeAbsent && oursShape != theirsShape:
			from, shape := theirs, theirsShape
			if winner == Ours {
				from, shape = ours, oursShape
			}
			switch shape {
			case shapeScalar:
				scalars[k] = from.ExtraScalars[k]
			case shapeList:
				lists[k] = from.ExtraLists[k]
			case shapeBlock:
				blocks[k] = from.ExtraBlocks[k]
			}
		case oursShape == shapeList || theirsShape == shapeList:
			if merged := setMerge(base.ExtraLists[k], ours.ExtraLists[k], theirs.ExtraLists[k]); merged != nil {
				lists[k] = merged
			}
		case oursShape == shapeBlock || theirsShape == shapeBlock:
			if merged := threeWayPtr(mapEntry(base.ExtraBlocks, k), mapEntry(ours.ExtraBlocks, k), mapEntry(theirs.ExtraBlocks, k), winner); merged != nil {
				blocks[k] = *merged
			}
		default:
			if merged := threeWayPtr(mapEntry(base.ExtraScalars, k), mapEntry(ours.ExtraScalars, k), mapEntry(theirs.ExtraScalars, k), winner); merged != nil {
				scalars[k] = *merged
			}
		}
	}

	return nilIfEmpty(scalars), nilIfEmpty(lists), nilIfEmpty(blocks)
}

func nilIfEmpty[V any](m map[string]V) map[string]V {
	if len(m) == 0 {
		return nil
	}
	return m
}
