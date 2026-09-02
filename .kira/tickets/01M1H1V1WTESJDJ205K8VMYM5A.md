---
id: 01M1H1V1WTESJDJ205K8VMYM5A
number: CORE-59
aliases: []
type: ticket
title: "An epic can be given an epic parent by create --parent, which link forbids"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-09-02T14:34:57+02:00
updated: 2026-09-02T14:34:57+02:00
---
Three parts of the tool disagree about whether an epic may have an epic
parent. Verified on a scratch repo (`kira init --key TST`):

    kira link TST-2 --epic TST-1
      -> refused: "invalid item: an epic cannot have an epic parent"
         (internal/core/link.go:62)

    kira create epic "epic via parent" --parent TST-1
      -> accepted. The file is written with `epic: <ulid of TST-1>` on an
         item of type epic, and `kira show TST-3` prints `epic: TST-1`.

    kira list --tree
      -> renders the nested epic at top level, silently ignoring the parent
         link: internal/core/list.go:295 treats every epic as a group root
         (`if r.Type == TypeEpic { ensureGroup(r.ID); continue }`).

    kira doctor
      -> "ok: 0 errors, 0 warnings" on a repo holding state that `link`
         declares invalid.

So the invariant is enforced on one write path, violated by another, ignored
by the tree renderer, and invisible to doctor.

`internal/core/progress.go` recurses through child epics and carries cycle
detection (`errx.Conflict("epic cycle detected at %s")`), which implies
nesting was expected to be reachable. So the question is which side is
right, not simply where to add a guard.

Decide the intended model first:

- epics never nest — then `create --parent` must refuse it, and the
  recursion plus cycle detection in progress.go is unreachable and can go;
- epics may nest — then link.go:62 is the bug and `list --tree` should nest
  them.

Either way doctor should report the state it currently passes silently.

Context: this is one of nine sites in `internal/core` that special-case
`datamodel.TypeEpic` (create.go:121, list.go:295,343, link.go:62,
progress.go:47,66,78,79, board.go:55). All nine encode a single capability —
"this type parents other items and rolls up their progress" — which is a
candidate for a schema-declared flag rather than a hardcoded type
comparison, in the same spirit as CORE-55.
