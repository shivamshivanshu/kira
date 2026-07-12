// Package item implements the codec for kira items: markdown files with
// block-style YAML frontmatter. Parse and Serialize are inverses on canonical
// files (byte-stable round-trip), and a single field edit produces a single
// changed frontmatter line. See docs/design/02-data-model.md (schema, comment
// grammar) and docs/design/03-storage-and-git.md (writer invariants).
package item

import (
	"fmt"
	"strings"
	"time"
)

// Item is the typed frontmatter schema. Field declaration order is the
// canonical serialization order (docs/design/02-data-model.md §1); do not
// reorder without updating the writer, which relies on it.
//
// Optional scalar fields (priority, owner, reporter, estimate) are pointers:
// nil means the key is absent and the writer omits its line entirely. epic is
// required but nullable, so a nil Epic is still written, as `epic: null`.
type Item struct {
	ID        string
	Number    string
	Aliases   []string
	Type      string
	Title     string
	State     string
	Priority  *string
	Owner     *string
	Reporter  *string
	Labels    []string
	Epic      *string
	BlockedBy []string
	Estimate  *float64
	Created   string
	Updated   string

	// Body is the markdown after the closing frontmatter fence, stored and
	// re-emitted verbatim. Comments live here; use ParseComments/AppendComment.
	Body string
}

// The two legal values of the type field.
const (
	TypeTicket = "ticket"
	TypeEpic   = "epic"
)

// Frontmatter key names, in canonical order.
const (
	keyID        = "id"
	keyNumber    = "number"
	keyAliases   = "aliases"
	keyType      = "type"
	keyTitle     = "title"
	keyState     = "state"
	keyPriority  = "priority"
	keyOwner     = "owner"
	keyReporter  = "reporter"
	keyLabels    = "labels"
	keyEpic      = "epic"
	keyBlockedBy = "blocked_by"
	keyEstimate  = "estimate"
	keyCreated   = "created"
	keyUpdated   = "updated"
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
