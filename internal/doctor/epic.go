package doctor

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
)

func EpicCycles(items []*datamodel.Item, resolver *id.Resolver) []Finding {
	byID := map[string]*datamodel.Item{}
	for _, it := range items {
		if it.ID != "" {
			byID[it.ID] = it
		}
	}
	parent := parentEdges(byID, resolver)

	done := map[string]bool{}
	var out []Finding
	for _, it := range items {
		if it.ID == "" || done[it.ID] {
			continue
		}
		out = append(out, walkForCycle(it.ID, parent, byID, done)...)
	}
	return out
}

func parentEdges(byID map[string]*datamodel.Item, resolver *id.Resolver) map[string]string {
	parent := map[string]string{}
	for ulid, it := range byID {
		if it.Epic == nil || *it.Epic == "" {
			continue
		}
		target := *it.Epic
		if resolver != nil {
			if resolved, err := resolver.Resolve(*it.Epic); err == nil {
				target = resolved
			}
		}
		if _, ok := byID[target]; ok {
			parent[ulid] = target
		}
	}
	return parent
}

func walkForCycle(start string, parent map[string]string, byID map[string]*datamodel.Item, done map[string]bool) []Finding {
	var path []string
	index := map[string]int{}
	cur := start
	for {
		if at, seen := index[cur]; seen {
			cycle := path[at:]
			markDone(path, done)
			return cycleFindings(cycle, byID)
		}
		if done[cur] {
			markDone(path, done)
			return nil
		}
		index[cur] = len(path)
		path = append(path, cur)
		next, ok := parent[cur]
		if !ok {
			markDone(path, done)
			return nil
		}
		cur = next
	}
}

func markDone(path []string, done map[string]bool) {
	for _, n := range path {
		done[n] = true
	}
}

func cycleFindings(cycle []string, byID map[string]*datamodel.Item) []Finding {
	trail := strings.Join(numbersOf(cycle, byID), " -> ")
	out := make([]Finding, 0, len(cycle))
	for _, ulid := range cycle {
		out = append(out, Finding{
			Class:    ClassCycle,
			Severity: SeverityError,
			ItemID:   ulid,
			Number:   numberOf(ulid, byID),
			Field:    datamodel.KeyEpic,
			Message:  "epic chain forms a cycle: " + trail,
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
