// Package reconcile computes deterministic ID-collision repair plans over a
// snapshot of items, independent of git, storage, or the doctor command.
package reconcile

import (
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/id"
)

type Renumber struct {
	ULID string
	From string
	To   string
}

func Plan(snap id.Snapshot) []Renumber {
	liveHolders := map[string][]string{}
	for _, it := range snap.Items {
		liveHolders[it.Number] = append(liveHolders[it.Number], it.ULID)
	}

	collided := make([]string, 0)
	for number, holders := range liveHolders {
		if len(holders) > 1 {
			collided = append(collided, number)
		}
	}
	slices.Sort(collided)

	next := map[string]int{}
	nextFor := func(key string) int {
		norm := strings.ToUpper(key)
		n, ok := next[norm]
		if !ok {
			n = id.HighestN(snap, key) + 1
		}
		next[norm] = n + 1
		return n
	}
	var plan []Renumber
	for _, number := range collided {
		holders := append([]string(nil), liveHolders[number]...)
		slices.Sort(holders)
		key := snap.Key
		if num, err := id.ParseNumber(number); err == nil {
			key = num.Key
		}
		for _, ulid := range holders[1:] {
			to := id.Number{Key: key, N: nextFor(key)}.String()
			plan = append(plan, Renumber{ULID: ulid, From: number, To: to})
		}
	}
	return plan
}
