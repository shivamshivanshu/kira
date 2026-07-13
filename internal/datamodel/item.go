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
}

const (
	TypeTicket = "ticket"
	TypeEpic   = "epic"
)

func ValidType(t string) bool { return t == TypeTicket || t == TypeEpic }

const ResolutionDropped = "dropped"

const (
	LinkRelates     = "relates"
	LinkDuplicateOf = "duplicate_of"
)

var LinkTypes = []string{LinkRelates, LinkDuplicateOf}

func ValidLinkType(t string) bool { return slices.Contains(LinkTypes, t) }

func ValidDate(s string) bool {
	_, err := time.Parse(time.DateOnly, s)
	return err == nil
}

var MutableFields = []string{
	KeySubtype, KeyTitle, KeyResolution, KeyPriority, KeyRank, KeyOwner,
	KeyReporter, KeyLabels, KeyEpic, KeySprint, KeyDue, KeyEstimate,
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

var FrontmatterKeys = []string{
	KeyID, KeyNumber, KeyAliases, KeyType, KeySubtype, KeyTitle, KeyState,
	KeyResolution, KeyPriority, KeyRank, KeyOwner, KeyReporter, KeyLabels,
	KeyEpic, KeyBlockedBy, KeyLinks, KeySprint, KeyDue, KeyEstimate,
	KeyCreated, KeyUpdated,
}

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
