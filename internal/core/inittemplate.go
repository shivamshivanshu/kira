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

commit:
  mode: auto                  # auto | manual | prompt
  trailer: Kira-Ticket        # trailer key that links a commit to its ticket
  close_trailer: Kira-Closes  # trailer key that closes a ticket when the commit lands

merge:
  policy: auto                # auto | manual: auto-resolve same-field kira conflicts vs leave markers

sync:
  push: false                 # true = kira sync publishes after a clean pull; default is pull-only

workon:
  branch_pattern: "{key}/{number}-{slug}"   # tokens: {key} {number} {slug}
  casing: kebab               # kebab | snake: casing of {slug} in the branch name

ui:
  icons: auto                 # auto | nerd | emoji | text — if glyphs render as boxes, set emoji or text (nerd font not installed)
  background: auto            # auto | dark | light: palette the TUI and board assume

estimate:
  unit: points                # points | hours
  hours_per_day: 8            # working hours per day, for the hours↔days rollups

# --- optional blocks: uncomment and edit to enable (defaults shown are inert) ---

# filters: named saved queries, run via kira list --filter <name>. docs: docs/design/02-data-model.md
#filters: {}

# sprints: sprint entities, created by kira sprint create; items key into this list. docs: docs/design/02-data-model.md
#sprints:

# git: relate kira items to the commits and branches that touch them. docs: docs/design/07-git-integration.md
#   landed_ref — ref (e.g. origin/main) whose history marks work as merged/landed
#   scan_since — limit commit scanning to history after this ref or date
#git: {}
`

func initConfigYAML(key, name string) string {
	return fmt.Sprintf(configTemplate, key, name)
}
