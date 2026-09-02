// Package setx provides small, dependency-free set and dedup helpers shared
// across the codebase.
package setx

import "slices"

// Deduper tracks which keys have been seen. Add reports whether k is new.
type Deduper[K comparable] struct {
	seen map[K]bool
}

func NewDeduper[K comparable]() *Deduper[K] {
	return &Deduper[K]{seen: map[K]bool{}}
}

// Has answers without marking, for a caller that must not claim k as its own.
func (d *Deduper[K]) Has(k K) bool { return d.seen[k] }

func (d *Deduper[K]) Add(k K) bool {
	if d.seen[k] {
		return false
	}
	d.seen[k] = true
	return true
}

func ToSet[T comparable](xs []T) map[T]bool {
	s := make(map[T]bool, len(xs))
	for _, x := range xs {
		s[x] = true
	}
	return s
}

// AppendUnique returns xs unchanged if it already contains v, else xs with v appended.
func AppendUnique[T comparable](xs []T, v T) []T {
	if slices.Contains(xs, v) {
		return xs
	}
	return append(xs, v)
}
