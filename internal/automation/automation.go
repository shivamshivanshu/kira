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

	ItemID string
	Number string
	Type   string
	Title  string
	Commit string
}
