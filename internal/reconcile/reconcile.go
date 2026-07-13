// Package reconcile computes deterministic ID-collision repair plans over a
// snapshot of items, independent of git, storage, or the doctor command.
package reconcile

import (
	"sort"

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
	sort.Strings(collided)

	next := id.Allocate(snap).N
	var plan []Renumber
	for _, number := range collided {
		holders := append([]string(nil), liveHolders[number]...)
		sort.Strings(holders)
		for _, ulid := range holders[1:] {
			to := id.Number{Key: snap.Key, N: next}.String()
			next++
			plan = append(plan, Renumber{ULID: ulid, From: number, To: to})
		}
	}
	return plan
}
