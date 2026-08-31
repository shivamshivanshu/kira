package webui

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/setx"
)

type bulkOutcome struct {
	Ref, Number, Message, Error string
}

func (s *server) bulk(cfg *datamodel.Config, f url.Values) ([]bulkOutcome, error) {
	refs := f["ids"]
	if len(refs) == 0 || len(refs) > 100 {
		return nil, errx.User("select between 1 and 100 items")
	}
	operation, value := f.Get("operation"), f.Get("value")
	switch operation {
	case "move", "assign", "label-add", "label-rm", "board-move", "sprint":
	default:
		return nil, errx.User("unknown bulk operation %q", operation)
	}
	if value == "" && operation != "assign" && operation != "sprint" {
		return nil, errx.User("a value is required for %s", operation)
	}
	batch, err := s.store.BeginBatch(cfg)
	if err != nil {
		return nil, err
	}
	defer batch.Close()
	outcomes, err := resolveBulk(batch, refs)
	if err != nil {
		return nil, err
	}
	if len(outcomes) == 0 {
		return nil, errx.User("select at least one item")
	}
	for i := range outcomes {
		row := &outcomes[i]
		if row.Error != "" {
			continue
		}
		message, number, err := bulkItem(batch, operation, row.Ref, value, f.Get("resolution"))
		if number != "" {
			row.Number = number
		}
		if err != nil {
			row.Error = userError(err)
		} else {
			row.Message = message
		}
	}
	return outcomes, nil
}

func resolveBulk(batch *core.Batch, refs []string) ([]bulkOutcome, error) {
	seen := setx.NewDeduper[string]()
	outcomes := make([]bulkOutcome, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" || !seen.Add(ref) {
			continue
		}
		it, err := batch.Resolve(ref)
		if err != nil {
			outcomes = append(outcomes, bulkOutcome{Ref: ref, Error: userError(err)})
			continue
		}
		if it.ID != ref && !seen.Add(it.ID) {
			continue
		}
		outcomes = append(outcomes, bulkOutcome{Ref: it.ID, Number: it.Number})
	}
	return outcomes, nil
}

func bulkItem(batch *core.Batch, operation, ref, value, resolution string) (string, string, error) {
	switch operation {
	case "move":
		res, err := batch.Move(ref, value, core.MoveOpts{Resolution: resolution})
		if err != nil {
			return "", "", err
		}
		return joinMessages("Moved to "+res.To, strings.Join(res.Warnings, "\n")), res.Number, nil
	case "assign":
		res, err := batch.Assign(ref, value, core.AssignOpts{})
		return mutationMessage(res, err, "Assigned", "")
	case "label-add", "label-rm":
		res, err := batch.LabelSet(ref, value, operation == "label-add", false)
		return mutationMessage(res, err, "Updated labels for", "")
	case "board-move":
		res, err := batch.BoardMove(ref, value)
		if err != nil {
			return "", "", err
		}
		return "Moved " + res.From + " to " + res.To, res.To, nil
	case "sprint":
		res, err := batch.EditFields(ref, []core.FieldEdit{{Key: "sprint", Value: value}}, false)
		return mutationMessage(res, err, "Updated sprint for", "")
	default:
		return "", "", errx.User("unknown bulk operation %q", operation)
	}
}

func bulkMessage(outcomes []bulkOutcome) string {
	failed := 0
	for _, outcome := range outcomes {
		if outcome.Error != "" {
			failed++
		}
	}
	return fmt.Sprintf("Bulk action: %d succeeded, %d failed.", len(outcomes)-failed, failed)
}

func bulkRetry(f url.Values, outcomes []bulkOutcome) url.Values {
	retry := url.Values{"operation": {f.Get("operation")}, "value": {f.Get("value")}, "resolution": {f.Get("resolution")}}
	for _, outcome := range outcomes {
		if outcome.Error != "" {
			retry.Add("ids", outcome.Ref)
		}
	}
	return retry
}
