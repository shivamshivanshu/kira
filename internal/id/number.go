package id

import (
	"fmt"
	"strconv"
	"strings"
)

// Number represents a display number in the form KEY-n.
type Number struct {
	Key string
	N   int
}

// String returns the display number as a formatted KEY-n string.
func (n Number) String() string { return fmt.Sprintf("%s-%d", n.Key, n.N) }

func splitLastDash(s string) (key, suffix string, ok bool) {
	i := strings.LastIndexByte(s, '-')
	if i <= 0 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

// KeyOf extracts the KEY portion from a display number string.
func KeyOf(number string) string {
	key, _, _ := splitLastDash(number)
	return key
}

func ParseNumber(s string) (Number, error) {
	key, suffix, ok := splitLastDash(s)
	if !ok || suffix == "" {
		return Number{}, fmt.Errorf("id: %q is not a KEY-n number", s)
	}
	n, err := strconv.Atoi(suffix)
	if err != nil || n <= 0 {
		return Number{}, fmt.Errorf("id: %q has a non-positive-integer suffix", s)
	}
	return Number{Key: key, N: n}, nil
}

// HighestN returns the highest integer suffix among all numbers and aliases for a given key in the snapshot.
func HighestN(snap Snapshot, key string) int {
	highest := 0
	bump := func(raw string) {
		if num, err := ParseNumber(raw); err == nil && strings.EqualFold(num.Key, key) && num.N > highest {
			highest = num.N
		}
	}
	for _, it := range snap.Items {
		bump(it.Number)
		for _, a := range it.Aliases {
			bump(a)
		}
	}
	for _, n := range snap.Reserved {
		bump(n)
	}
	return highest
}

// Allocator generates sequential display numbers for one board key.
type Allocator struct {
	key  string
	next int
}

func NewAllocator(snap Snapshot, key string) *Allocator {
	return &Allocator{key: key, next: HighestN(snap, key) + 1}
}

func (a *Allocator) Alloc() string {
	number := Number{Key: a.key, N: a.next}.String()
	a.next++
	return number
}
