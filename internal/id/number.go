package id

import (
	"fmt"
	"strconv"
	"strings"
)

// Number is a parsed sequential display number, e.g. {Key: "KIRA", N: 142}.
// Hash-style numbers (see HashNumber) are not sequential and are handled as
// opaque strings, so they are never represented as a Number.
type Number struct {
	Key string
	N   int
}

// String renders the canonical KEY-n form.
func (n Number) String() string { return fmt.Sprintf("%s-%d", n.Key, n.N) }

// ParseNumber parses a KEY-n display number. The key is compared case-
// insensitively by callers but preserved as written here; the suffix must be a
// positive integer. A bare integer (no key) is rejected — resolution, not
// parsing, is where a bare number is completed against the project key.
func ParseNumber(s string) (Number, error) {
	i := strings.LastIndex(s, "-")
	if i <= 0 || i == len(s)-1 {
		return Number{}, fmt.Errorf("id: %q is not a KEY-n number", s)
	}
	n, err := strconv.Atoi(s[i+1:])
	if err != nil || n <= 0 {
		return Number{}, fmt.Errorf("id: %q has a non-positive-integer suffix", s)
	}
	return Number{Key: s[:i], N: n}, nil
}

// Allocate returns the next sequential number for snap's project: max(N)+1 over
// the uniqueness domain — the union of every item's live number and every
// item's aliases (02-data-model §7). A retired number sitting in aliases still
// reserves its slot, so the result never collides with a live or retired value.
// This same computation serves both first allocation and doctor collision
// repair. Only entries whose key matches snap.Key (case-insensitively) count;
// unparseable or foreign-key entries are ignored. An empty domain yields N=1.
func Allocate(snap Snapshot) Number {
	highest := 0
	bump := func(raw string) {
		if num, err := ParseNumber(raw); err == nil && strings.EqualFold(num.Key, snap.Key) && num.N > highest {
			highest = num.N
		}
	}
	for _, it := range snap.Items {
		bump(it.Number)
		for _, a := range it.Aliases {
			bump(a)
		}
	}
	return Number{Key: snap.Key, N: highest + 1}
}

// HashNumber derives a display number directly from the ULID instead of
// allocating one (id.style: hash — 02-data-model §7). Being a pure function of
// the ULID, it needs no counter, no post-merge reconciliation, and grows no
// aliases. The suffix is the trailing six characters of the canonical ULID.
func HashNumber(key string, u ULID) string {
	s := u.String()
	return key + "-" + s[len(s)-6:]
}
