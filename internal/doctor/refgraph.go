package doctor

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
)

func RefCycles(items []*datamodel.Item, resolver *id.Resolver) []Finding {
	byID := itemsByID(items)
	var out []Finding
	for _, rel := range datamodel.CycleRelations() {
		edges := func(it *datamodel.Item) []string { return resolveRefs(rel.Edges(it), resolver, byID) }
		for _, cycle := range datamodel.FindCycles(byID, edges) {
			out = append(out, cyclePathFindings(ClassRefCycle, rel.Field, cycle, byID)...)
		}
	}
	return out
}

func NonEpicParents(items []*datamodel.Item, resolver *id.Resolver) []Finding {
	byID := itemsByID(items)
	var out []Finding
	for _, it := range items {
		if it.ID == "" || it.Epic == nil || *it.Epic == "" {
			continue
		}
		target := resolveRef(*it.Epic, resolver)
		parent := byID[target]
		if parent == nil || parent.Type == datamodel.TypeEpic {
			continue
		}
		out = append(out, Finding{
			Class:    ClassEpicKind,
			Severity: SeverityError,
			ItemID:   it.ID,
			Number:   it.Number,
			Field:    datamodel.KeyEpic,
			Message:  "epic points at " + numberOf(target, byID) + ", which is not an epic",
		})
	}
	return out
}

func itemsByID(items []*datamodel.Item) map[string]*datamodel.Item {
	byID := make(map[string]*datamodel.Item, len(items))
	for _, it := range items {
		if it.ID != "" {
			byID[it.ID] = it
		}
	}
	return byID
}

func resolveRef(ref string, resolver *id.Resolver) string {
	if resolver != nil {
		if ulid, err := resolver.Resolve(ref); err == nil {
			return ulid
		}
	}
	return ref
}

func resolveRefs(refs []string, resolver *id.Resolver, byID map[string]*datamodel.Item) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		if target := resolveRef(r, resolver); byID[target] != nil {
			out = append(out, target)
		}
	}
	return out
}
