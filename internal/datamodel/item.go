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

	CRLF bool
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

type ContainerProgress struct {
	Done  int
	Total int
}
