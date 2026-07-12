package id

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oklog/ulid/v2"
)

type Item struct {
	ULID    string
	Number  string
	Aliases []string
}

type Snapshot struct {
	Key   string
	Items []Item
}

type NotFoundError struct{ Token string }

func (e *NotFoundError) Error() string { return fmt.Sprintf("id: %q resolves to no item", e.Token) }

type AmbiguousError struct {
	Token      string
	Candidates []string
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("id: prefix %q is ambiguous between %s", e.Token, strings.Join(e.Candidates, ", "))
}

type Resolver struct {
	key          string
	sortedULIDs  []string
	ulidByNumber map[string]string
}

func NewResolver(snap Snapshot) *Resolver {
	r := &Resolver{
		key:          snap.Key,
		sortedULIDs:  make([]string, len(snap.Items)),
		ulidByNumber: make(map[string]string, len(snap.Items)),
	}
	for i, it := range snap.Items {
		r.sortedULIDs[i] = strings.ToUpper(it.ULID)
		for _, a := range it.Aliases {
			r.ulidByNumber[strings.ToUpper(a)] = r.sortedULIDs[i]
		}
	}
	for i, it := range snap.Items {
		r.ulidByNumber[strings.ToUpper(it.Number)] = r.sortedULIDs[i]
	}
	sort.Strings(r.sortedULIDs)
	return r
}

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
	if u, ok := r.ulidByNumber[numKey]; ok {
		return u, nil
	}

	return "", &NotFoundError{Token: token}
}

func (r *Resolver) contains(u string) bool {
	i := sort.SearchStrings(r.sortedULIDs, u)
	return i < len(r.sortedULIDs) && r.sortedULIDs[i] == u
}

func (r *Resolver) prefixMatches(up string) []string {
	if len(up) >= ulid.EncodedSize {
		return nil
	}
	var out []string
	for _, u := range r.sortedULIDs[sort.SearchStrings(r.sortedULIDs, up):] {
		if !strings.HasPrefix(u, up) {
			break
		}
		out = append(out, u)
	}
	return out
}
