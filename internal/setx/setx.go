// Package setx provides small, dependency-free set and dedup helpers shared
// across the codebase.
package setx

// Deduper tracks which keys have been seen. Add reports whether k is new.
type Deduper[K comparable] struct {
	seen map[K]bool
}

func NewDeduper[K comparable]() *Deduper[K] {
	return &Deduper[K]{seen: map[K]bool{}}
}

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
