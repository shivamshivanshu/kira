package doctor

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
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
	trail := strings.Join(numbersOf(cycle, byID), " -> ")
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

func numbersOf(ulids []string, byID map[string]*datamodel.Item) []string {
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
