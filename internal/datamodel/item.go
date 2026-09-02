package datamodel

import (
	"slices"
	"time"

	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/setx"
)

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
	Parent     *string
	BlockedBy  []string
	Links      map[string][]string
	Sprint     *string
	Due        *string
	Estimate   *float64
	Created    string
	Updated    string

	Body string

	Activity string

	// ExtraScalars and ExtraLists hold frontmatter as it sits on disk -- the
	// rendered YAML text, quoting and type intact, so `points: 42` survives a
	// rewrite unquoted. Read them through codec.Decode*, write them through
	// codec.Encode*; comparing one against a raw value silently mismatches.
	// ExtraLinks is the exception: link targets are bare refs, stored raw.
	ExtraScalars map[string]Rendered
	ExtraLists   map[string][]Rendered
	ExtraLinks   map[string][]string
	CRLF         bool
}

// Rendered is frontmatter text exactly as it sits on disk, quoting and YAML
// type intact. It is a distinct type so the compiler refuses to let it stand in
// for a decoded value: passing one to something expecting the value behind it
// is the mistake that caused every extras bug so far.
type Rendered string

func (it *Item) ExtraKeys() []string {
	out := make([]string, 0, len(it.ExtraScalars)+len(it.ExtraLists))
	for k := range it.ExtraScalars {
		out = append(out, k)
	}
	for k := range it.ExtraLists {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// ChangedExtraKeys is the extras-map counterpart to ChangedFields, kept
// separate because ChangedFields callers feed its output to schema lookups
// that must see only built-in keys.
func ChangedExtraKeys(a, b *Item) []string {
	dedup := setx.NewDeduper[string]()
	var out []string
	for _, k := range append(a.ExtraKeys(), b.ExtraKeys()...) {
		if !dedup.Add(k) {
			continue
		}
		if a.ExtraScalars[k] != b.ExtraScalars[k] || !slices.Equal(a.ExtraLists[k], b.ExtraLists[k]) {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

// IndexByID maps items by ID, skipping any with an empty ID.
func IndexByID(items []*Item) map[string]*Item {
	m := make(map[string]*Item, len(items))
	for _, it := range items {
		if it.ID != "" {
			m[it.ID] = it
		}
	}
	return m
}

const (
	TypeTicket = "ticket"
	TypeEpic   = "epic"
)

const ResolutionDropped = "dropped"

type LinkType string

const (
	LinkRelates     LinkType = "relates"
	LinkDuplicateOf LinkType = "duplicate_of"
)

var LinkTypes = []LinkType{LinkRelates, LinkDuplicateOf}

func ValidLinkType(t string) bool { return slices.Contains(LinkTypes, LinkType(t)) }

func CloneStringListMap[T ~string](m map[string][]T) map[string][]T {
	if m == nil {
		return nil
	}
	out := make(map[string][]T, len(m))
	for k, v := range m {
		out[k] = slices.Clone(v)
	}
	return out
}

func ValidDate(s string) bool {
	_, err := time.Parse(time.DateOnly, s)
	return err == nil
}

const (
	KeyID         = "id"
	KeyNumber     = "number"
	KeyAliases    = "aliases"
	KeyType       = "type"
	KeySubtype    = "subtype"
	KeyTitle      = "title"
	KeyState      = "state"
	KeyResolution = "resolution"
	KeyPriority   = "priority"
	KeyRank       = "rank"
	KeyOwner      = "owner"
	KeyReporter   = "reporter"
	KeyLabels     = "labels"
	KeyParent     = "parent"
	KeyBlockedBy  = "blocked_by"
	KeyLinks      = "links"
	KeySprint     = "sprint"
	KeyDue        = "due"
	KeyEstimate   = "estimate"
	KeyCreated    = "created"
	KeyUpdated    = "updated"
	KeyBody       = "body"
)

var FrontmatterKeys = []string{
	KeyID, KeyNumber, KeyAliases, KeyType, KeySubtype, KeyTitle, KeyState,
	KeyResolution, KeyPriority, KeyRank, KeyOwner, KeyReporter, KeyLabels,
	KeyParent, KeyBlockedBy, KeyLinks, KeySprint, KeyDue, KeyEstimate,
	KeyCreated, KeyUpdated,
}

var frontmatterKeySet = setx.ToSet(FrontmatterKeys)

func IsFrontmatterKey(k string) bool { return frontmatterKeySet[k] }

func (it *Item) CreatedTime() (time.Time, error) { return time.Parse(time.RFC3339, it.Created) }

func (it *Item) UpdatedTime() (time.Time, error) { return time.Parse(time.RFC3339, it.Updated) }

type ParseError struct {
	Errs []error
}

func (e *ParseError) Error() string {
	return errx.JoinErrors("invalid item", e.Errs)
}

func (e *ParseError) Unwrap() []error { return e.Errs }

type Comment struct {
	ID     string
	Author string
	Ts     string
	Body   string
}

type EpicProgress struct {
	Done  int
	Total int
}
