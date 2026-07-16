// Package id implements kira's two-tier item identity: immutable ULIDs plus renumberable KIRA-n display numbers.
package id

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

type ULID = ulid.ULID

var entropy = &ulid.LockedMonotonicReader{
	MonotonicReader: ulid.Monotonic(rand.Reader, 0),
}

func Mint() ULID { return mint(time.Now()) }

func mint(t time.Time) ULID { return ulid.MustNew(ulid.Timestamp(t), entropy) }

func ParseULID(s string) (ULID, error) { return ulid.ParseStrict(s) }

// ParseULIDLoose reports whether s is ULID-shaped (26 chars, no timestamp
// overflow) without ParseULID's strict base32 charset check — used to
// classify filenames as item files, where test fixtures commonly use
// human-readable near-ULID strings that aren't strictly valid encodings.
func ParseULIDLoose(s string) (ULID, error) { return ulid.Parse(s) }
