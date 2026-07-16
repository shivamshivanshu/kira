// Package automation runs user-defined observer scripts post-commit; it is named automation, not hooks, because internal/hooks and .kira/hooks/ are git-hook territory.
package automation

import (
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/syncx"
)

const PayloadVersion = 1

const RecursionGuardEnv = "KIRA_AUTOMATION"

type Actor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Change struct {
	Old string `json:"old"`
	New string `json:"new"`
}

type Event struct {
	Name         datamodel.EventName
	Source       datamodel.ChangeSource
	Item         *datamodel.ShowResult
	Changes      map[string]Change
	From         string
	To           string
	FromCategory string
	ToCategory   string
	Sync         *syncx.Report
	Commit       string
}

func (e Event) itemID() string {
	if e.Item == nil {
		return ""
	}
	return e.Item.ID
}

func (e Event) itemNumber() string {
	if e.Item == nil {
		return ""
	}
	return e.Item.Number
}

func (e Event) itemType() string {
	if e.Item == nil {
		return ""
	}
	return e.Item.Type
}

func (e Event) itemTitle() string {
	if e.Item == nil {
		return ""
	}
	return e.Item.Title
}
