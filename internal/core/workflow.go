package core

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

func UnknownTypeError(typ string) error {
	return errx.User("%q is not a known type", typ).WithHint("known types: %s", strings.Join(datamodel.ItemTypes, ", "))
}

func stateKeys(wf datamodel.Workflow) []string {
	out := make([]string, 0, len(wf.States))
	for _, s := range wf.States {
		out = append(out, s.Key)
	}
	return out
}

func ItemMoveTargets(cfg *datamodel.Config, typ, subtype string) []string {
	workflow, _ := cfg.ItemWorkflow(typ, subtype)
	return stateKeys(workflow)
}

func WorkflowStateNames(cfg *datamodel.Config, typ string) []string {
	names := ItemMoveTargets(cfg, typ, "")
	if cfg != nil && typ == datamodel.TypeTicket {
		for _, subtype := range datamodel.Subtypes {
			for _, state := range cfg.TicketStates[subtype] {
				names = append(names, state.Key)
			}
		}
	}
	return names
}

func workflowFor(cfg *datamodel.Config, it *datamodel.Item) datamodel.Workflow {
	workflow, _ := cfg.ItemWorkflow(it.Type, ptr.Deref(it.Subtype))
	return workflow
}

func sameWorkflowState(a, b *datamodel.Item) bool {
	return a != nil && a.Type == b.Type && ptr.Equal(a.Subtype, b.Subtype) && a.State == b.State
}

func itemIsDone(cfg *datamodel.Config, it *datamodel.Item) bool {
	category, _ := cfg.ItemCategory(it)
	return category == datamodel.CategoryDone
}

func itemCategoryString(cfg *datamodel.Config, it *datamodel.Item) string {
	category, _ := cfg.ItemCategory(it)
	return string(category)
}

func clearStaleResolution(cfg *datamodel.Config, it *datamodel.Item) {
	if category, known := cfg.ItemCategory(it); known && category != datamodel.CategoryDone {
		it.Resolution = nil
	}
}

func validateMergedWorkflow(cfg *datamodel.Config, item, ours, theirs *datamodel.Item) error {
	if _, declared := workflowFor(cfg, item).StateByKey(item.State); declared || sameWorkflowState(ours, item) || sameWorkflowState(theirs, item) {
		return nil
	}
	return errx.Conflict("%s: state %q is outside the merged ticket's workflow", item.Number, item.State)
}
