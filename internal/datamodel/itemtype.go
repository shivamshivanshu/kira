package datamodel

const (
	TypeTicket = "ticket"
	TypeEpic   = "epic"
)

var ItemTypes = []string{TypeEpic, TypeTicket}

func IsContainer(typ string) bool { return typ == TypeEpic }

func WorkflowOf(typ string) (Workflow, bool) {
	switch typ {
	case TypeTicket:
		return Workflow{
			States: []State{
				{Key: "TODO", Category: CategoryTodo},
				{Key: "IN_PROGRESS", Category: CategoryDoing, Wip: 3},
				{Key: "REVIEW", Category: CategoryDoing, Wip: 2},
				{Key: "DONE", Category: CategoryDone, Resolution: "done"},
				{Key: "WONT_DO", Category: CategoryDone, Resolution: ResolutionDropped},
			},
			Initial: "TODO",
		}, true
	case TypeEpic:
		return Workflow{
			States: []State{
				{Key: "PLANNED", Category: CategoryTodo},
				{Key: "ACTIVE", Category: CategoryDoing},
				{Key: "DONE", Category: CategoryDone, Resolution: "done"},
			},
			Initial: "PLANNED",
		}, true
	}
	return Workflow{}, false
}
