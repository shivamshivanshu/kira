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
  # known: [bug, feature, perf, tech-debt, infra]   # example vocabulary
  strict: false

people:
  known: []                   # add known people; strict rejects unknown assignees
  # known: [{ name: jdoe }, { name: asmith }]        # example roster
  strict: false

priorities: [P0, P1, P2, P3]  # ordered high→low: validates priority, defines ranked sort
                              # and >/< query compare; [] = free-form (equality-only)
subtypes:    [bug, story, task, spike]   # validates subtype; [] = free-form
resolutions: [done, dropped, duplicate, cannot-reproduce]  # validates resolution; [] = free-form

commit:
  mode: auto                  # auto | manual | prompt (prompt needs a terminal; without one, changes stage but are not committed)
  trailer: Kira-Ticket        # trailer key that links a commit to its ticket
  close_trailer: Kira-Closes  # trailer key that closes a ticket when the commit lands
  link_markers: [trailer, subject, leading_number]  # what links a commit: trailer, a [[KIRA-n]] subject marker, and/or a bare KIRA-n leading the subject
  reference_markers: [bare]   # what weakly references a commit: bare KIRA-n refs; [] to disable

merge:
  policy: auto                # auto | manual: auto-resolve same-field kira conflicts vs leave markers

sync:
  push: false                 # true = kira sync publishes after a clean pull; default is pull-only

# workon: branch and worktree naming (built-in defaults shown; uncomment a key to pin it for this repo)
#   branch_pattern: "{key}/{number}-{slug}"   # tokens: {key} {number} {slug}
#   casing: kebab             # kebab | snake: casing of {slug} in the branch name

# ui: TUI and board appearance (personal presentation; prefer ~/.config/kira/config.yaml so pins are not shared across clones)
#   icons: auto               # auto | nerd | emoji | text
#   background: auto          # auto | dark | light

estimate:
  unit: points                # points | hours

# --- optional blocks: uncomment and edit to enable (defaults shown are inert) ---

# filters: named saved queries, run via kira list --filter <name>.
filters:
  next: state = TODO AND NOT blocked ORDER BY priority

# sprints: sprint entities, created by kira sprint create; items key into this list.
#sprints:

# git: relate kira items to the commits and branches that touch them.
#   landed_ref — ref (e.g. origin/main) whose history marks work as merged/landed
#   scan_since — limit commit scanning to history after this ref or date
#git: {}
`

func initConfigYAML(key, name string) string {
	return fmt.Sprintf(configTemplate, key, name)
}
