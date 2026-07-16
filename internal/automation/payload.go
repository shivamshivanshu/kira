package automation

import (
	"bytes"
	"encoding/json"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/syncx"
)

// HookPayload is the JSON written to an automation hook's stdin.
type HookPayload struct {
	PayloadVersion int                    `json:"payload_version"`
	Event          datamodel.EventName    `json:"event"`
	Source         datamodel.ChangeSource `json:"source,omitempty"`
	Ts             string                 `json:"ts"`
	Repo           string                 `json:"repo"`
	Actor          Actor                  `json:"actor"`
	Item           *datamodel.ShowResult  `json:"item,omitempty"`
	Changes        map[string]Change      `json:"changes,omitempty"`
	From           string                 `json:"from,omitempty"`
	To             string                 `json:"to,omitempty"`
	ToCategory     string                 `json:"to_category,omitempty"`
	FromCategory   string                 `json:"from_category,omitempty"`
	Commit         string                 `json:"commit,omitempty"`
	Sync           *syncx.Report          `json:"sync,omitempty"`
}

func Payload(ev Event, repo, ts string, actor Actor) ([]byte, error) {
	p := HookPayload{
		PayloadVersion: PayloadVersion,
		Event:          ev.Name,
		Source:         ev.Source,
		Ts:             ts,
		Repo:           repo,
		Actor:          actor,
		Changes:        ev.Changes,
		From:           ev.From,
		To:             ev.To,
		ToCategory:     ev.ToCategory,
		FromCategory:   ev.FromCategory,
		Commit:         ev.Commit,
		Sync:           ev.Sync,
		Item:           ev.Item,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
