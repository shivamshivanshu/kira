package doctor

import (
	"fmt"
	"maps"
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func Collisions(items []*datamodel.Item) []Finding {
	live := map[string][]string{}
	alias := map[string][]string{}
	for _, it := range items {
		if it.ID == "" {
			continue
		}
		if it.Number != "" {
			live[it.Number] = appendUnique(live[it.Number], it.ID)
		}
		for _, a := range it.Aliases {
			if a != "" {
				alias[a] = appendUnique(alias[a], it.ID)
			}
		}
	}

	values := map[string]bool{}
	for v := range live {
		values[v] = true
	}
	for v := range alias {
		values[v] = true
	}

	var out []Finding
	for _, value := range slices.Sorted(maps.Keys(values)) {
		liveIDs := live[value]
		aliasIDs := without(alias[value], liveIDs)
		switch {
		case len(liveIDs) >= 2:
			out = append(out, collisionFinding(value, CollisionLiveLive, liveIDs, aliasIDs, slices.Min(liveIDs)))
		case len(liveIDs) == 1 && len(aliasIDs) >= 1:
			out = append(out, collisionFinding(value, CollisionLiveAlias, liveIDs, aliasIDs, liveIDs[0]))
		case len(liveIDs) == 0 && len(aliasIDs) >= 2:
			out = append(out, collisionFinding(value, CollisionAliasAlias, liveIDs, aliasIDs, slices.Min(aliasIDs)))
		}
	}
	return out
}

func collisionFinding(value string, kind CollisionKind, liveIDs, aliasIDs []string, keep string) Finding {
	live := sortedIDs(liveIDs)
	alias := sortedIDs(aliasIDs)
	return Finding{
		Class:    ClassCollision,
		Severity: SeverityError,
		Number:   value,
		Message:  fmt.Sprintf("number %s is claimed by %d items (%s)", value, len(live)+len(alias), kind),
		Collision: &Collision{
			Value:    value,
			Kind:     kind,
			LiveIDs:  live,
			AliasIDs: alias,
			Keep:     keep,
		},
	}
}

func sortedIDs(xs []string) []string {
	out := slices.Sorted(slices.Values(xs))
	if out == nil {
		return []string{}
	}
	return out
}

func appendUnique(xs []string, more ...string) []string {
	for _, m := range more {
		if !slices.Contains(xs, m) {
			xs = append(xs, m)
		}
	}
	return xs
}

func without(xs, remove []string) []string {
	var out []string
	for _, x := range xs {
		if !slices.Contains(remove, x) {
			out = append(out, x)
		}
	}
	return out
}
