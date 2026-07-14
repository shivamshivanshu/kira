package id_test

import (
	"slices"
	"testing"

	"github.com/shivamshivanshu/kira/internal/id"
)

func TestSortKeyOrdersNumericallyThenLexicallyThenByULID(t *testing.T) {
	t.Parallel()
	keys := []id.SortKey{
		id.NewSortKey("KIRA-10", uB),
		id.NewSortKey("KIRA-2", uA),
		id.NewSortKey("KIRA-2", u1),
		id.NewSortKey("ZED-1", u1),
		id.NewSortKey("no-number-here", u1),
		id.NewSortKey("", uA),
	}
	slices.SortFunc(keys, func(a, b id.SortKey) int {
		switch {
		case a.Less(b):
			return -1
		case b.Less(a):
			return 1
		default:
			return 0
		}
	})
	want := []string{"", "ZED-1", "KIRA-2", "KIRA-2", "KIRA-10", "no-number-here"}
	for i, k := range keys {
		if k.Number != want[i] {
			t.Fatalf("order[%d] = %q, want %q (full order %v)", i, k.Number, want[i], keys)
		}
	}
	if keys[2].ULID != u1 || keys[3].ULID != uA {
		t.Fatalf("equal numbers must tie-break on ULID: got %q then %q", keys[2].ULID, keys[3].ULID)
	}
}

func TestSortKeyParsesNumber(t *testing.T) {
	t.Parallel()
	if k := id.NewSortKey("KIRA-142", u1); !k.OK || k.N != 142 {
		t.Fatalf("NewSortKey(KIRA-142) = %+v, want OK with N=142", k)
	}
	if k := id.NewSortKey("garbage", u1); k.OK {
		t.Fatalf("NewSortKey(garbage) = %+v, want !OK", k)
	}
}
