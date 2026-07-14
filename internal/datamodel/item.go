package datamodel

import (
	"fmt"
	"slices"
	"strings"
	"time"
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
	Epic       *string
	BlockedBy  []string
	Links      map[string][]string
	Sprint     *string
	Due        *string
	Estimate   *float64
	Created    string
	Updated    string

	Body string

	UnknownKeys      []string `json:"-"`
	UnknownLinkTypes []string `json:"-"`
	CRLF             bool     `json:"-"`
}

func (it *Item) HasUnknown() bool {
	return len(it.UnknownKeys) > 0 || len(it.UnknownLinkTypes) > 0
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
	KeyEpic       = "epic"
	KeyBlockedBy  = "blocked_by"
	KeyLinks      = "links"
	KeySprint     = "sprint"
	KeyDue        = "due"
	KeyEstimate   = "estimate"
	KeyCreated    = "created"
	KeyUpdated    = "updated"
	KeyBody       = "body"
)

type itemKey struct {
	key         string
	frontmatter bool
	mutable     bool
}

var itemKeys = []itemKey{
	{KeyID, true, false},
	{KeyNumber, true, false},
	{KeyAliases, true, false},
	{KeyType, true, false},
	{KeySubtype, true, true},
	{KeyTitle, true, true},
	{KeyState, true, false},
	{KeyResolution, true, true},
	{KeyPriority, true, true},
	{KeyRank, true, true},
	{KeyOwner, true, true},
	{KeyReporter, true, true},
	{KeyLabels, true, true},
	{KeyEpic, true, true},
	{KeyBlockedBy, true, false},
	{KeyLinks, true, false},
	{KeySprint, true, true},
	{KeyDue, true, true},
	{KeyEstimate, true, true},
	{KeyCreated, true, false},
	{KeyUpdated, true, false},
	{KeyBody, false, false},
}

func selectItemKeys(pred func(itemKey) bool) []string {
	out := make([]string, 0, len(itemKeys))
	for _, k := range itemKeys {
		if pred(k) {
			out = append(out, k.key)
		}
	}
	return out
}

var FrontmatterKeys = selectItemKeys(func(k itemKey) bool { return k.frontmatter })

var MutableFields = selectItemKeys(func(k itemKey) bool { return k.mutable })

var frontmatterKeySet = func() map[string]bool {
	m := make(map[string]bool, len(FrontmatterKeys))
	for _, k := range FrontmatterKeys {
		m[k] = true
	}
	return m
}()

func IsFrontmatterKey(k string) bool { return frontmatterKeySet[k] }

func (it *Item) CreatedTime() (time.Time, error) { return time.Parse(time.RFC3339, it.Created) }

func (it *Item) UpdatedTime() (time.Time, error) { return time.Parse(time.RFC3339, it.Updated) }

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
