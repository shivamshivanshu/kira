package doctor

import (
	"fmt"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
)

func cyclePathFindings(class Class, field string, cycle []string, byID map[string]*datamodel.Item) []Finding {
	trail := strings.Join(cycleTrail(cycle, byID), " -> ")
	out := make([]Finding, 0, len(cycle))
	for _, ulid := range cycle {
		out = append(out, Finding{
			Class:    class,
			Severity: SeverityError,
			ItemID:   ulid,
			Number:   numberOf(ulid, byID),
			Field:    field,
			Message:  field + " chain forms a cycle: " + trail,
		})
	}
	return out
}

func cycleTrail(ulids []string, byID map[string]*datamodel.Item) []string {
	out := make([]string, len(ulids))
	for i, u := range ulids {
		out[i] = numberOf(u, byID)
	}
	return append(out, out[0])
}

func numberOf(ulid string, byID map[string]*datamodel.Item) string {
	if it := byID[ulid]; it != nil && it.Number != "" {
		return it.Number
	}
	return ulid
}

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

func RefTypeMismatches(items []*datamodel.Item, resolver *id.Resolver) []Finding {
	byID := itemsByID(items)
	var out []Finding
	for _, it := range items {
		if it.ID == "" || it.Parent == nil {
			continue
		}
		target := resolveRef(*it.Parent, resolver)
		parent := byID[target]
		if parent == nil || datamodel.IsContainer(parent.Type) {
			continue
		}
		out = append(out, Finding{
			Class:    ClassRefType,
			Severity: SeverityError,
			ItemID:   it.ID,
			Number:   it.Number,
			Field:    datamodel.KeyParent,
			Message:  fmt.Sprintf("%s points at %s of type %s, which does not group other items", datamodel.KeyParent, numberOf(target, byID), parent.Type),
		})
	}
	return out
}

func itemsByID(items []*datamodel.Item) map[string]*datamodel.Item {
	return datamodel.IndexByID(items)
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
