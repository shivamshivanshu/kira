package core

import "fmt"

// configTemplate is the commented config.yaml init writes. It mirrors the
// documented default (docs/design/02-data-model.md §9) with two deliberate
// differences: the project key/name are the caller's, and labels.known /
// people.known are seeded empty (a fresh project has no vocabulary yet). Every
// omitted field inherits config.Default() via config.Parse, so this stays the
// human-editable surface, not the exhaustive schema. Pinned by an init unit
// test that round-trips it through config.Parse.
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
      - { key: IN_PROGRESS, category: doing }
      - { key: REVIEW,      category: doing }
      - { key: DONE,        category: done }
      - { key: WONT_DO,     category: done, resolution: dropped }
    initial: TODO
    transitions:
      TODO:        [IN_PROGRESS, WONT_DO]
      IN_PROGRESS: [REVIEW, TODO, WONT_DO]
      REVIEW:      [DONE, IN_PROGRESS]
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
