package id

import (
	"fmt"
	"strconv"
	"strings"
)

type Number struct {
	Key string
	N   int
}

func (n Number) String() string { return fmt.Sprintf("%s-%d", n.Key, n.N) }

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

func HashNumber(key string, u ULID) string {
	s := u.String()
	return key + "-" + s[len(s)-6:]
}

func AllocFor(hashStyle bool, key string, next int, u ULID) string {
	if hashStyle {
		return HashNumber(key, u)
	}
	return Number{Key: key, N: next}.String()
}
