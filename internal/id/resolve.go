package id

import (
	"fmt"
	"slices"
	"strings"

	"github.com/oklog/ulid/v2"

	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/setx"
)

// Item represents an item in a snapshot with its ULID, display number, and aliases.
type Item struct {
	ULID    string
	Number  string
	Aliases []string
}

// Snapshot represents a set of items for resolution purposes.
type Snapshot struct {
	Key   string
	Items []Item
}

// NotFoundError indicates that a token does not resolve to any item.
type NotFoundError struct {
	Token      string
	Suggestion string
}

// Error returns the error message for NotFoundError.
func (e *NotFoundError) Error() string { return fmt.Sprintf("id: %q resolves to no item", e.Token) }

// AmbiguousError indicates that a token resolves to multiple items.
type AmbiguousError struct {
	Token      string
	Candidates []string
}

// Error returns the error message for AmbiguousError.
func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("%q is ambiguous between %s", e.Token, strings.Join(e.Candidates, ", "))
}

// Resolver resolves item identifiers (ULIDs, display numbers, or aliases) to ULIDs.
type Resolver struct {
	sortedULIDs      []string
	liveByNumber     map[string][]string
	aliasHolders     map[string][]string
	liveNumberByULID map[string]string
	numbers          []string
	numberByBareN    map[string][]string
}

// NewResolver creates a Resolver from a snapshot.
func NewResolver(snap Snapshot) *Resolver {
	r := &Resolver{
		sortedULIDs:      make([]string, len(snap.Items)),
		liveByNumber:     make(map[string][]string, len(snap.Items)),
		aliasHolders:     map[string][]string{},
		liveNumberByULID: make(map[string]string, len(snap.Items)),
		numbers:          make([]string, len(snap.Items)),
		numberByBareN:    map[string][]string{},
	}
	addBare := func(up string) {
		if _, bare, ok := splitLastDash(up); ok {
			if !slices.Contains(r.numberByBareN[bare], up) {
				r.numberByBareN[bare] = append(r.numberByBareN[bare], up)
			}
		}
	}
	for i, it := range snap.Items {
		u := strings.ToUpper(it.ULID)
		live := strings.ToUpper(it.Number)
		r.sortedULIDs[i] = u
		r.numbers[i] = live
		r.liveByNumber[live] = append(r.liveByNumber[live], u)
		r.liveNumberByULID[u] = live
		addBare(live)
		for _, a := range it.Aliases {
			up := strings.ToUpper(a)
			if !slices.Contains(r.aliasHolders[up], u) {
				r.aliasHolders[up] = append(r.aliasHolders[up], u)
			}
			addBare(up)
		}
	}
	slices.Sort(r.sortedULIDs)
	return r
}

func (r *Resolver) holdersOf(full string) []string {
	if h, ok := r.liveByNumber[full]; ok {
		return h
	}
	return r.aliasHolders[full]
}

func (r *Resolver) ambiguous(token string, holders []string) error {
	dedup := setx.NewDeduper[string]()
	var cands []string
	for _, u := range holders {
		if n := r.liveNumberByULID[u]; dedup.Add(n) {
			cands = append(cands, n)
		}
	}
	if len(cands) < len(holders) {
		cands = slices.Clone(holders)
	}
	slices.Sort(cands)
	return &AmbiguousError{Token: token, Candidates: cands}
}

// Resolve returns the ULID for the given identifier token (ULID, display number, or alias).
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

	if strings.Contains(up, "-") {
		h := r.holdersOf(up)
		switch len(h) {
		case 0:
			return "", &NotFoundError{Token: token, Suggestion: errx.Nearest(up, r.numbers)}
		case 1:
			return h[0], nil
		default:
			return "", r.ambiguous(token, h)
		}
	}
	return r.resolveBare(token, up)
}

func (r *Resolver) resolveBare(token, up string) (string, error) {
	var holders []string
	for _, f := range r.numberByBareN[up] {
		for _, u := range r.holdersOf(f) {
			if !slices.Contains(holders, u) {
				holders = append(holders, u)
			}
		}
	}
	switch len(holders) {
	case 0:
		return "", &NotFoundError{Token: token, Suggestion: errx.Nearest(up, r.numbers)}
	case 1:
		return holders[0], nil
	default:
		return "", r.ambiguous(token, holders)
	}
}

func (r *Resolver) contains(u string) bool {
	_, found := slices.BinarySearch(r.sortedULIDs, u)
	return found
}

func (r *Resolver) prefixMatches(up string) []string {
	if len(up) >= ulid.EncodedSize {
		return nil
	}
	var out []string
	start, _ := slices.BinarySearch(r.sortedULIDs, up)
	for _, u := range r.sortedULIDs[start:] {
		if !strings.HasPrefix(u, up) {
			break
		}
		out = append(out, u)
	}
	return out
}
