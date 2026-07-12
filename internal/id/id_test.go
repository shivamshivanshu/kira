package id

import (
	"strings"
	"testing"
	"time"
)

func assertStrictlyIncreasingUnique(t *testing.T, ids []ULID) {
	t.Helper()
	seen := make(map[string]struct{}, len(ids))
	for i, u := range ids {
		s := u.String()
		if _, dup := seen[s]; dup {
			t.Fatalf("duplicate ULID minted: %s", s)
		}
		seen[s] = struct{}{}
		if i > 0 && ids[i-1].Compare(u) >= 0 {
			t.Fatalf("not strictly increasing at %d: %s !< %s", i, ids[i-1], u)
		}
	}
}

func TestMintUniqueAndMonotonic(t *testing.T) {
	const n = 5000
	ids := make([]ULID, n)
	for i := range ids {
		ids[i] = Mint()
	}
	assertStrictlyIncreasingUnique(t, ids)
}

func TestMintMonotonicWithinOneMillisecond(t *testing.T) {
	fixed := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	const n = 5000
	ids := make([]ULID, n)
	for i := range ids {
		ids[i] = mint(fixed)
	}
	assertStrictlyIncreasingUnique(t, ids)
}

func TestParseULIDCaseInsensitive(t *testing.T) {
	canonical := "01AN4Z07BY79KA1307SR9X4MV3"
	for _, in := range []string{canonical, strings.ToLower(canonical)} {
		u, err := ParseULID(in)
		if err != nil {
			t.Fatalf("ParseULID(%q) error: %v", in, err)
		}
		if got := u.String(); got != canonical {
			t.Fatalf("ParseULID(%q).String() = %q, want %q", in, got, canonical)
		}
	}
}

func TestParseULIDRejectsInvalid(t *testing.T) {
	cases := map[string]string{
		"too short":           "01AN4Z07BY",
		"too long":            "01AN4Z07BY79KA1307SR9X4MV3X",
		"ambiguous letter I":  "01AN4Z07BY79KA1307SR9X4MVI",
		"ambiguous letter O":  "01AN4Z07BY79KA1307SR9X4MVO",
		"non-base32":          "01AN4Z07BY79KA1307SR9X4M!3",
		"overflow first char": "81AN4Z07BY79KA1307SR9X4MV3",
	}
	for name, in := range cases {
		if _, err := ParseULID(in); err == nil {
			t.Errorf("%s: ParseULID(%q) succeeded, want error", name, in)
		}
	}
}
