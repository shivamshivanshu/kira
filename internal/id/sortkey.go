package id

// SortKey represents a sortable item key for ordering items by number or ULID.
type SortKey struct {
	N      int
	OK     bool
	Number string
	ULID   string
}

// NewSortKey creates a SortKey from a display number and ULID string.
func NewSortKey(number, ulid string) SortKey {
	k := SortKey{Number: number, ULID: ulid}
	if n, err := ParseNumber(number); err == nil {
		k.N, k.OK = n.N, true
	}
	return k
}

// Less reports whether k should sort before o.
func (k SortKey) Less(o SortKey) bool {
	if k.OK && o.OK && k.N != o.N {
		return k.N < o.N
	}
	if k.Number != o.Number {
		return k.Number < o.Number
	}
	return k.ULID < o.ULID
}
