package core

import "fmt"

const configTemplate = `version: 1

project:
  key: %s
  name: %s

id:
  style: sequential          # sequential (default, reconciled post-merge) | hash (ULID-derived, zero reconciliation)

workflows:
  ticket:
    states:
      - { key: TODO,        category: todo }
      - { key: IN_PROGRESS, category: doing, wip: 3 }   # advisory column cap (0/absent = unlimited)
      - { key: REVIEW,      category: doing, wip: 2 }
      - { key: DONE,        category: done }
      - { key: WONT_DO,     category: done, resolution: dropped }
    initial: TODO
    transitions:
      TODO:        [IN_PROGRESS, WONT_DO]
      IN_PROGRESS: [REVIEW, TODO, WONT_DO]
      REVIEW:                                            # bare strings and guard maps may mix
        - { to: DONE, require: [resolution], set: { resolution: done } }
        - IN_PROGRESS
      DONE:        []
      WONT_DO:     []
    enforce_transitions: true
  epic:
    states:
      - { key: PLANNED, category: todo }
      - { key: ACTIVE,  category: doing }
      - { key: DONE,    category: done }
    initial: PLANNED
    transitions:
      PLANNED: [ACTIVE]
      ACTIVE:  [DONE]
      DONE:    []

labels:
  known: []                   # add project labels; strict rejects unknown ones (--force overrides)
  strict: false

people:
  known: []
  strict: false

priorities: [P0, P1, P2, P3]  # ordered high→low: validates priority, defines ranked sort
                              # and >/< query compare; [] = free-form (equality-only)
subtypes:    [bug, story, task, spike]   # validates subtype; [] = free-form
resolutions: [done, dropped, duplicate, cannot-reproduce]  # validates resolution; [] = free-form

filters: {}                   # named saved queries -> kira list --filter <name>

sprints: []                   # sprint entities (kira sprint create); item sprint keys into this

commit:
  mode: auto                  # auto | manual | prompt
  trailer: Kira-Ticket

merge:
  policy: auto                # auto | manual

sync:
  push: false

ui:
  icons: auto                 # auto | always | never

estimate:
  unit: points                # points | hours
`

func initConfigYAML(key, name string) string {
	return fmt.Sprintf(configTemplate, key, name)
}
