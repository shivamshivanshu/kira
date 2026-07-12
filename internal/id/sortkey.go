package id

type SortKey struct {
	N      int
	OK     bool
	Number string
	ULID   string
}

func NewSortKey(number, ulid string) SortKey {
	k := SortKey{Number: number, ULID: ulid}
	if n, err := ParseNumber(number); err == nil {
		k.N, k.OK = n.N, true
	}
	return k
}

func (k SortKey) Less(o SortKey) bool {
	if k.OK && o.OK && k.N != o.N {
		return k.N < o.N
	}
	if k.Number != o.Number {
		return k.Number < o.Number
	}
	return k.ULID < o.ULID
}
