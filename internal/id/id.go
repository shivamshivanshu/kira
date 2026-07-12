// Package id implements kira's two-tier item identity: immutable ULIDs plus renumberable KIRA-n display numbers.
package id

import (
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

type ULID = ulid.ULID

var entropy = &ulid.LockedMonotonicReader{
	MonotonicReader: ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0),
}

func Mint() ULID { return mint(time.Now()) }

func mint(t time.Time) ULID { return ulid.MustNew(ulid.Timestamp(t), entropy) }

func ParseULID(s string) (ULID, error) { return ulid.ParseStrict(s) }
