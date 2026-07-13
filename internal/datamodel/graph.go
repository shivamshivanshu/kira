package datamodel

import (
	"slices"
	"strings"
)

type Relation struct {
	Field string
	Edges func(*Item) []string
}

func CycleRelations() []Relation {
	return []Relation{
		{Field: KeyBlockedBy, Edges: func(it *Item) []string { return it.BlockedBy }},
		{Field: string(LinkDuplicateOf), Edges: func(it *Item) []string { return it.Links[string(LinkDuplicateOf)] }},
	}
}

func FindCycles(byID map[string]*Item, edges func(*Item) []string) [][]string {
	const (
		white = iota
		gray
		black
	)
	color := make(map[string]int, len(byID))
	seen := map[string]bool{}
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
					if key := cycleKey(cyc); key != "" && !seen[key] {
						seen[key] = true
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
