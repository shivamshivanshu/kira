// Package item implements the codec for kira items: markdown files with
// block-style YAML frontmatter. Parse and Serialize are inverses on canonical
// files (byte-stable round-trip), and a single field edit produces a single
// changed frontmatter line. See docs/design/02-data-model.md (schema, comment
// grammar) and docs/design/03-storage-and-git.md (writer invariants).
package item

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// Item is the typed frontmatter schema. Field declaration order is the
// canonical serialization order (docs/design/02-data-model.md §1); do not
// reorder without updating the writer, which relies on it.
//
// Optional scalar fields (subtype, resolution, priority, rank, sprint, due,
// owner, reporter, estimate) are pointers: nil means the key is absent and the
// writer omits its line entirely. epic is required but nullable, so a nil Epic
// is still written, as `epic: null`. Links is nil when absent; when present it
// holds only known link types with non-empty target lists (the canonical form —
// an empty list is the same as an absent type).
type Item struct {
	ID         string
	Number     string
	Aliases    []string
	Type       string
	Subtype    *string
	Title      string
	State      string
	Resolution *string
	Priority   *string
	Rank       *string
	Owner      *string
	Reporter   *string
	Labels     []string
	Epic       *string
	BlockedBy  []string
	Links      map[string][]string
	Sprint     *string
	Due        *string
	Estimate   *float64
	Created    string
	Updated    string

	// Body is the markdown after the closing frontmatter fence, stored and
	// re-emitted verbatim. Comments live here; use ParseComments/AppendComment.
	Body string
}

// The two legal values of the type field.
const (
	TypeTicket = "ticket"
	TypeEpic   = "epic"
)

// ValidType reports whether t is one of the two legal item types. It is the one
// home of the ticket|epic rule, shared by the parser and core's validation.
func ValidType(t string) bool { return t == TypeTicket || t == TypeEpic }

// The v1 typed link types (docs/design/02-data-model.md §3).
const (
	LinkRelates     = "relates"
	LinkDuplicateOf = "duplicate_of"
)

// LinkTypes is the one home of the known link types: it drives parsing,
// canonical emission order, the JSON view shape, and the CLI's per-type flags.
var LinkTypes = []string{LinkRelates, LinkDuplicateOf}

// ValidLinkType reports whether t is a known v1 link type.
func ValidLinkType(t string) bool { return slices.Contains(LinkTypes, t) }

// ValidDate reports whether s is a valid RFC3339 full date (the `due` and
// config sprint start/end format).
func ValidDate(s string) bool {
	_, err := time.Parse(time.DateOnly, s)
	return err == nil
}

// MutableFields are the user-mutable frontmatter field names, in canonical
// order — the schema surface a config transition guard (require:/set:) may
// name. Excluded deliberately: state (the transition itself owns it — a guard
// attaches to a move, it does not perform one) and the cross-reference lists
// (blocked_by, links: guards assign scalar-ish values, not edges).
var MutableFields = []string{
	keySubtype, keyTitle, keyResolution, keyPriority, keyRank, keyOwner,
	keyReporter, keyLabels, keyEpic, keySprint, keyDue, keyEstimate,
}

// Frontmatter key names, in canonical order.
const (
	keyID         = "id"
	keyNumber     = "number"
	keyAliases    = "aliases"
	keyType       = "type"
	keySubtype    = "subtype"
	keyTitle      = "title"
	keyState      = "state"
	keyResolution = "resolution"
	keyPriority   = "priority"
	keyRank       = "rank"
	keyOwner      = "owner"
	keyReporter   = "reporter"
	keyLabels     = "labels"
	keyEpic       = "epic"
	keyBlockedBy  = "blocked_by"
	keyLinks      = "links"
	keySprint     = "sprint"
	keyDue        = "due"
	keyEstimate   = "estimate"
	keyCreated    = "created"
	keyUpdated    = "updated"
)

// CreatedTime parses the Created timestamp. Created is stored as its exact
// original text (RFC3339) so it round-trips byte-for-byte; this accessor is the
// typed view of it.
func (it *Item) CreatedTime() (time.Time, error) { return time.Parse(time.RFC3339, it.Created) }

// UpdatedTime parses the Updated timestamp. See CreatedTime.
func (it *Item) UpdatedTime() (time.Time, error) { return time.Parse(time.RFC3339, it.Updated) }

// ParseError aggregates every validation failure found while parsing one item,
// rather than stopping at the first. Errs is the full list; Unwrap exposes it
// to errors.Is/As.
type ParseError struct {
	Errs []error
}

func (e *ParseError) Error() string {
	msgs := make([]string, len(e.Errs))
	for i, err := range e.Errs {
		msgs[i] = err.Error()
	}
	return fmt.Sprintf("invalid item: %s", strings.Join(msgs, "; "))
}

// Unwrap returns the wrapped errors for errors.Is/As traversal (Go 1.20+).
func (e *ParseError) Unwrap() []error { return e.Errs }
