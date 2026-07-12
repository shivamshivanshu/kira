package id

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oklog/ulid/v2"
)

// Item is the identity projection of one stored item — the only fields
// resolution and allocation need. The storage layer builds these from
// tickets/*.md frontmatter.
type Item struct {
	ULID    string
	Number  string
	Aliases []string
}

// Snapshot is the set of known items plus the project key, the input to
// Allocate and NewResolver.
type Snapshot struct {
	Key   string
	Items []Item
}

// NotFoundError reports that a token matched no item under any resolution rule.
type NotFoundError struct{ Token string }

func (e *NotFoundError) Error() string { return fmt.Sprintf("id: %q resolves to no item", e.Token) }

// AmbiguousError reports that a ULID prefix matched more than one item. The
// prefix is never silently resolved to one of them (02-data-model §7);
// Candidates lists the colliding ULIDs, sorted.
type AmbiguousError struct {
	Token      string
	Candidates []string
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("id: prefix %q is ambiguous between %s", e.Token, strings.Join(e.Candidates, ", "))
}

// Resolver resolves user-supplied item references against a fixed snapshot. It
// precomputes its lookup tables once, so a single Resolver should be reused
// across the many references a command resolves.
type Resolver struct {
	key     string
	ulids   []string          // canonical, sorted; the sole source of ULID identity
	numbers map[string]string // canonical number/alias -> ULID
}

// NewResolver builds a Resolver from snap. ULIDs are canonicalized to uppercase;
// numbers and aliases are indexed so both current and retired display numbers
// resolve. Aliases are inserted before live numbers so that, if a live number
// collides with another item's alias, the live number wins.
func NewResolver(snap Snapshot) *Resolver {
	r := &Resolver{
		key:     snap.Key,
		ulids:   make([]string, len(snap.Items)),
		numbers: make(map[string]string, len(snap.Items)),
	}
	for i, it := range snap.Items {
		r.ulids[i] = strings.ToUpper(it.ULID)
		for _, a := range it.Aliases {
			r.numbers[strings.ToUpper(a)] = r.ulids[i]
		}
	}
	for i, it := range snap.Items {
		r.numbers[strings.ToUpper(it.Number)] = r.ulids[i]
	}
	sort.Strings(r.ulids)
	return r
}

// Resolve maps a user token to a canonical ULID, trying in the fixed order of
// 02-data-model §7: full ULID, then unique ULID prefix, then display number
// (KEY-n or bare n, current or alias). A prefix matching multiple items is a
// hard AmbiguousError; an unmatched token is a NotFoundError.
func (r *Resolver) Resolve(token string) (string, error) {
	t := strings.TrimSpace(token)
	if t == "" {
		return "", &NotFoundError{Token: token}
	}
	up := strings.ToUpper(t)

	if len(up) == ulid.EncodedSize {
		if _, err := ParseULID(up); err == nil {
			if r.contains(up) {
				return up, nil
			}
			return "", &NotFoundError{Token: token}
		}
	}

	cands := r.prefixMatches(up)
	if len(cands) == 1 {
		return cands[0], nil
	}
	if len(cands) > 1 {
		return "", &AmbiguousError{Token: token, Candidates: cands}
	}

	numKey := up
	if !strings.Contains(up, "-") {
		numKey = strings.ToUpper(r.key) + "-" + up
	}
	if u, ok := r.numbers[numKey]; ok {
		return u, nil
	}

	return "", &NotFoundError{Token: token}
}

// contains reports whether u is a known canonical ULID via binary search over
// the sorted slice that is the resolver's single source of ULID identity.
func (r *Resolver) contains(u string) bool {
	i := sort.SearchStrings(r.ulids, u)
	return i < len(r.ulids) && r.ulids[i] == u
}

// prefixMatches returns the canonical ULIDs that have up as a strict prefix.
// Matching ULIDs are contiguous in the sorted slice starting at the lower
// bound, so it seeks once and walks the run. A full-length token yields none —
// full ULIDs are handled by the exact path.
func (r *Resolver) prefixMatches(up string) []string {
	if len(up) >= ulid.EncodedSize {
		return nil
	}
	var out []string
	for _, u := range r.ulids[sort.SearchStrings(r.ulids, up):] {
		if !strings.HasPrefix(u, up) {
			break
		}
		out = append(out, u)
	}
	return out
}
