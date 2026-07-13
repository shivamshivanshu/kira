package automation

import (
	"bytes"
	"encoding/json"

	"github.com/shivamshivanshu/kira/internal/syncx"
)

type payload struct {
	PayloadVersion int               `json:"payload_version"`
	Event          string            `json:"event"`
	Source         string            `json:"source,omitempty"`
	Ts             string            `json:"ts"`
	Repo           string            `json:"repo"`
	Actor          Actor             `json:"actor"`
	Item           any               `json:"item,omitempty"`
	Changes        map[string]Change `json:"changes,omitempty"`
	From           string            `json:"from,omitempty"`
	To             string            `json:"to,omitempty"`
	ToCategory     string            `json:"to_category,omitempty"`
	Sync           *syncx.Report     `json:"sync,omitempty"`
}

func Payload(ev Event, repo, ts string, actor Actor) ([]byte, error) {
	p := payload{
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
		Sync:           ev.Sync,
	}
	if ev.Item != nil {
		p.Item = ev.Item
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func envMirror(ev Event, repo string) []string {
	return []string{
		RecursionGuardEnv + "=1",
		"KIRA_EVENT=" + ev.Name,
		"KIRA_ITEM=" + ev.ItemID,
		"KIRA_NUMBER=" + ev.Number,
		"KIRA_TYPE=" + ev.Type,
		"KIRA_TITLE=" + ev.Title,
		"KIRA_FROM=" + ev.From,
		"KIRA_TO=" + ev.To,
		"KIRA_TO_CATEGORY=" + ev.ToCategory,
		"KIRA_SOURCE=" + ev.Source,
		"KIRA_ROOT=" + repo,
		"KIRA_COMMIT=" + ev.Commit,
	}
}
