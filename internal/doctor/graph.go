package doctor

import (
	"fmt"
	"strings"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
	"github.com/shivamshivanshu/kira/internal/id"
)

func EpicCycles(items []*datamodel.Item, resolver *id.Resolver) []Finding {
	byID := itemsByID(items)
	edges := func(it *datamodel.Item) []string {
		if it.Parent == nil || *it.Parent == "" {
			return nil
		}
		return []string{resolveRef(*it.Parent, resolver)}
	}
	var out []Finding
	for _, cycle := range datamodel.FindCycles(byID, edges) {
		out = append(out, cyclePathFindings(ClassCycle, datamodel.KeyParent, cycle, byID)...)
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

func RefTypeMismatches(schemas map[string]entityschema.Schema, items []*datamodel.Item, resolver *id.Resolver, itemTypes map[string]bool) []Finding {
	byID := itemsByID(items)
	containers := entityschema.ContainerTypes(schemas)
	var out []Finding
	for _, it := range items {
		if it.ID == "" {
			continue
		}
		schema, ok := schemas[it.Type]
		if !ok {
			continue
		}
		values := entityschema.ProjectItem(schema, it, codec.Decoder{})
		for _, f := range schema.Fields {
			if !f.TargetsItemType(itemTypes) {
				continue
			}
			for _, ref := range refStrings(values[f.Name]) {
				target := resolveRef(ref, resolver)
				parent := byID[target]
				if parent == nil {
					continue
				}
				mismatch := ""
				switch {
				case f.TargetsContainer && !containers[parent.Type]:
					mismatch = fmt.Sprintf("%s points at %s of type %s, which does not group other items", f.Name, numberOf(target, byID), parent.Type)
				case f.Target != "" && !f.TargetsContainer && parent.Type != f.Target:
					mismatch = fmt.Sprintf("%s points at %s of type %s, expected %s", f.Name, numberOf(target, byID), parent.Type, f.Target)
				}
				if mismatch == "" {
					continue
				}
				out = append(out, Finding{
					Class:    ClassRefType,
					Severity: SeverityError,
					ItemID:   it.ID,
					Number:   it.Number,
					Field:    f.Name,
					Message:  mismatch,
				})
			}
		}
	}
	return out
}

func refStrings(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []string:
		return t
	}
	return nil
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
