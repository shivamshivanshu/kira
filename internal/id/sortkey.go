package id

// SortKey is a display item's decoded ordering key. Building it once per element
// (an O(n) pass) and comparing keys avoids re-parsing the KEY-n number — and,
// for hash-style numbers, re-hitting ParseNumber's error allocation — on every
// one of the O(n log n) comparisons a sort performs. It is the single home of
// kira's display order (docs/design/04-cli.md §7): by sequential number when
// both items are sequential, else by the raw number string, ties broken by ULID.
type SortKey struct {
	N      int    // sequential number, valid only when OK
	OK     bool   // the display number parsed as KEY-n
	Number string // raw display number, the fallback ordering
	ULID   string // final tiebreak
}

// NewSortKey builds the ordering key for a display number and its ULID.
func NewSortKey(number, ulid string) SortKey {
	k := SortKey{Number: number, ULID: ulid}
	if n, err := ParseNumber(number); err == nil {
		k.N, k.OK = n.N, true
	}
	return k
}

// Less reports whether k orders before o.
func (k SortKey) Less(o SortKey) bool {
	if k.OK && o.OK && k.N != o.N {
		return k.N < o.N
	}
	if k.Number != o.Number {
		return k.Number < o.Number
	}
	return k.ULID < o.ULID
}
