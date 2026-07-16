// Package id implements kira's two-tier item identity: immutable ULIDs plus renumberable KIRA-n display numbers.
package id

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// ULID is the type of kira's immutable item identifiers.
type ULID = ulid.ULID

// entropy is the shared monotonic random source for ULID generation.
var entropy = &ulid.LockedMonotonicReader{
	MonotonicReader: ulid.Monotonic(rand.Reader, 0),
}

// Mint returns a new ULID seeded from the current time.
func Mint() ULID { return mint(time.Now()) }

func mint(t time.Time) ULID { return ulid.MustNew(ulid.Timestamp(t), entropy) }

// ParseULID parses a ULID string with strict base32 charset validation.
func ParseULID(s string) (ULID, error) { return ulid.ParseStrict(s) }

// ParseULIDLoose reports whether s is ULID-shaped (26 chars, no timestamp
// overflow) without ParseULID's strict base32 charset check — used to
// classify filenames as item files, where test fixtures commonly use
// human-readable near-ULID strings that aren't strictly valid encodings.
func ParseULIDLoose(s string) (ULID, error) { return ulid.Parse(s) }
