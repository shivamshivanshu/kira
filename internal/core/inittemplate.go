package core

import "fmt"

const configTemplate = `version: 1

project:
  key: %s
  name: %s

labels:
  known: []                   # add project labels; strict rejects unknown ones (--force overrides)
  # known: [bug, feature, perf, tech-debt, infra]   # example vocabulary
  strict: false

people:
  known: []                   # add known people; strict rejects unknown assignees
  # known: [{ name: jdoe }, { name: asmith }]        # example roster
  strict: false

commit:
  mode: auto                  # auto | manual

merge:
  policy: auto                # auto | manual: auto-resolve same-field kira conflicts vs leave markers

sync:
  push: false                 # true = kira sync publishes after a clean pull; default is pull-only

# workon: branch and worktree naming (built-in defaults shown; uncomment a key to pin it for this repo)
#   branch_pattern: "{key}/{number}-{slug}"   # tokens: {key} {number} {slug}

# ui: TUI and board appearance (personal presentation; prefer ~/.config/kira/config.yaml so pins are not shared across clones)
#   icons: auto               # auto | nerd | emoji | text
#   background: dark          # dark | light

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
#git: {}
`

func initConfigYAML(key, name string) string {
	return fmt.Sprintf(configTemplate, key, name)
}
