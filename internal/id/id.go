// Package id implements kira's two-tier identity scheme (02-data-model §7):
// an immutable ULID that is the filename stem and the only value used in
// cross-references, plus a human display number (KIRA-n) that may be renumbered
// after a merge collision while old values survive forever as aliases.
//
// The package is pure: it never touches the filesystem or git. Allocation and
// resolution operate on a caller-supplied Snapshot of already-loaded items;
// scanning tickets/ into that Snapshot is the storage layer's job.
package id

import (
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// ULID is kira's immutable item identity. It is an alias for the underlying
// oklog type so callers get its String, Compare, and Time methods without
// importing oklog directly; time-sortability is what makes the collision
// tiebreak (07-git-integration §4) and creation-order sorting deterministic.
type ULID = ulid.ULID

// entropy is the process-wide monotonic entropy source. LockedMonotonicReader
// makes it safe for concurrent use; monotonic increments keep ULIDs minted in
// the same millisecond strictly increasing.
var entropy = &ulid.LockedMonotonicReader{
	MonotonicReader: ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0),
}

// Mint returns a fresh, strictly increasing ULID. It is safe for concurrent use.
func Mint() ULID { return mint(time.Now()) }

// mint is the clock seam behind Mint; tests drive it with a frozen time to
// prove monotonicity holds within a single millisecond.
func mint(t time.Time) ULID { return ulid.MustNew(ulid.Timestamp(t), entropy) }

// ParseULID validates s as a full ULID, accepting either case, and returns the
// decoded value. It rejects wrong-length input and any non-Crockford-base32
// character (unlike oklog's lenient Parse), so it is safe for validating
// hand-typed identity fields; call String on the result for the canonical
// uppercase form stored in cross-references.
func ParseULID(s string) (ULID, error) { return ulid.ParseStrict(s) }
