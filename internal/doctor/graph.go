package doctor

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/id"
)

func EpicCycles(items []*datamodel.Item, resolver *id.Resolver) []Finding {
	byID := itemsByID(items)
	edges := func(it *datamodel.Item) []string {
		if it.Epic == nil || *it.Epic == "" {
			return nil
		}
		return []string{resolveRef(*it.Epic, resolver)}
	}
	var out []Finding
	for _, cycle := range datamodel.FindCycles(byID, edges) {
		out = append(out, cyclePathFindings(ClassCycle, datamodel.KeyEpic, cycle, byID)...)
	}
	return out
}

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

// WrongTypeRefs reports refs whose target exists but is the wrong entity
// type; refFindings owns the dangling case. Ref fields carrying a source are
// config vocabulary, not items, so a resolver over items cannot see them.
func WrongTypeRefs(items []*datamodel.Item, resolver *id.Resolver, schemas map[string]entityschema.Schema) []Finding {
	byID := itemsByID(items)
	refs := entityschema.NewRefResolver(items)
	var out []Finding
	for _, it := range items {
		if it.ID == "" {
			continue
		}
		schema, ok := schemas[it.Type]
		if !ok {
			continue
		}
		for _, f := range schema.Fields {
			if f.Type != entityschema.FieldRef || f.Source != "" || f.Target == "" {
				continue
			}
			for _, ref := range refValues(it, f.Name) {
				target := resolveRef(ref, resolver)
				if byID[target] == nil || refs.Exists(f.Target, target) {
					continue
				}
				out = append(out, Finding{
					Class:    wrongTypeRefClass(f.Name),
					Severity: SeverityError,
					ItemID:   it.ID,
					Number:   it.Number,
					Field:    f.Name,
					Message:  f.Name + " points at " + numberOf(target, byID) + ", which is not " + indefinite(f.Target),
				})
			}
		}
	}
	return out
}

func wrongTypeRefClass(field string) Class {
	if field == datamodel.KeyEpic {
		return ClassEpicKind
	}
	return ClassRefKind
}

func refValues(it *datamodel.Item, field string) []string {
	if field == datamodel.KeyBlockedBy {
		return it.BlockedBy
	}
	d, ok := datamodel.Field(field)
	if !ok || d.Get == nil || d.Present == nil || !d.Present(it) {
		return nil
	}
	return []string{d.Get(it)}
}

func indefinite(noun string) string {
	if strings.ContainsAny(noun[:1], "aeiouAEIOU") {
		return "an " + noun
	}
	return "a " + noun
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
