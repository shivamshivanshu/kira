package id

import (
	"fmt"
	"strconv"
	"strings"
)

const hashSuffixLen = 6

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

// ParseNumber parses a display number string into a Number struct.
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
	return highest
}

// Allocate returns the next sequential number for the snapshot's key.
func Allocate(snap Snapshot) Number {
	return Number{Key: snap.Key, N: HighestN(snap, snap.Key) + 1}
}

func takenNumbers(snap Snapshot) map[string]bool {
	taken := make(map[string]bool, len(snap.Items))
	for _, it := range snap.Items {
		taken[strings.ToUpper(it.Number)] = true
		for _, a := range it.Aliases {
			taken[strings.ToUpper(a)] = true
		}
	}
	return taken
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Allocator generates display numbers either sequentially or by deriving from ULIDs.
type Allocator struct {
	hashStyle bool
	key       string
	next      int
	taken     map[string]bool
}

// NewAllocator creates an Allocator in either sequential or hash-based mode.
func NewAllocator(hashStyle bool, snap Snapshot, key string) *Allocator {
	a := &Allocator{hashStyle: hashStyle, key: key}
	if hashStyle {
		a.taken = takenNumbers(snap)
	} else {
		a.next = HighestN(snap, key) + 1
	}
	return a
}

// Alloc returns a display number for the given ULID.
func (a *Allocator) Alloc(u ULID) string {
	if !a.hashStyle {
		number := Number{Key: a.key, N: a.next}.String()
		a.next++
		return number
	}
	s := u.String()
	number := a.key + "-" + s
	for n := hashSuffixLen; n <= len(s); n++ {
		suffix := s[len(s)-n:]
		if !allDigits(suffix) && !a.taken[strings.ToUpper(a.key+"-"+suffix)] {
			number = a.key + "-" + suffix
			break
		}
	}
	a.taken[strings.ToUpper(number)] = true
	return number
}
