package datamodel

import (
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/setx"
)

type Relation struct {
	Field string
	Edges func(*Item) []string
}

// FindCycleFrom visits only nodes reachable from root and returns a cycle
// containing root. An unrelated pre-existing cycle does not block a mutation.
func FindCycleFrom(byID map[string]*Item, root string, edges func(*Item) []string) []string {
	seen := setx.NewDeduper[string]()
	var path []string
	var walk func(string) bool
	walk = func(id string) bool {
		it := byID[id]
		if it == nil || !seen.Add(id) {
			return false
		}
		path = append(path, id)
		for _, target := range edges(it) {
			if target == root || walk(target) {
				return true
			}
		}
		path = path[:len(path)-1]
		return false
	}
	if walk(root) {
		return path
	}
	return nil
}

func CycleRelations() []Relation {
	return []Relation{
		{Field: KeyBlockedBy, Edges: func(it *Item) []string { return it.BlockedBy }},
		{Field: string(LinkDuplicateOf), Edges: func(it *Item) []string { return it.Links[string(LinkDuplicateOf)] }},
		{Field: KeyParent, Edges: func(it *Item) []string {
			if it.Parent == nil || *it.Parent == "" {
				return nil
			}
			return []string{*it.Parent}
		}},
	}
}

func FindCycles(byID map[string]*Item, edges func(*Item) []string) [][]string {
	const (
		white = iota
		gray
		black
	)
	color := make(map[string]int, len(byID))
	dedup := setx.NewDeduper[string]()
	var stack []string
	var cycles [][]string

	var dfs func(u string)
	dfs = func(u string) {
		color[u] = gray
		stack = append(stack, u)
		if it := byID[u]; it != nil {
			for _, v := range edges(it) {
				if byID[v] == nil {
					continue
				}
				switch color[v] {
				case white:
					dfs(v)
				case gray:
					cyc := cycleFrom(stack, v)
					if key := cycleKey(cyc); key != "" && dedup.Add(key) {
						cycles = append(cycles, cyc)
					}
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[u] = black
	}

	ids := make([]string, 0, len(byID))
	for u := range byID {
		ids = append(ids, u)
	}
	slices.Sort(ids)
	for _, u := range ids {
		if color[u] == white {
			dfs(u)
		}
	}
	return cycles
}

func cycleFrom(stack []string, start string) []string {
	for i, n := range stack {
		if n == start {
			return slices.Clone(stack[i:])
		}
	}
	return nil
}

func cycleKey(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}
	lo := 0
	for i := range cycle {
		if cycle[i] < cycle[lo] {
			lo = i
		}
	}
	rotated := append(slices.Clone(cycle[lo:]), cycle[:lo]...)
	return strings.Join(rotated, ">")
}
