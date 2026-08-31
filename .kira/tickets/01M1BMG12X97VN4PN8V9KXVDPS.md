---
id: 01M1BMG12X97VN4PN8V9KXVDPS
number: KIRA-107
aliases: []
type: ticket
title: "Path traversal: unvalidated frontmatter id writes outside the store"
state: REVIEW
priority: P0
labels: []
epic: null
blocked_by: []
created: 2026-08-31T12:05:32+02:00
updated: 2026-08-31T12:13:21+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BMGKARTYM8YDNSWXAQ78WT author=Shivam-Shivanshu ts=2026-08-31T12:05:51+02:00 -->
Reproduced 2026-08-31 on main. codec.Parse (internal/codec/parse.go:45) reads 'id:' with reqScalar and never checks it is ULID-shaped; storage.ItemFilename (internal/storage/fs.go:70) is bare concat and ItemPath filepath.Joins it, so '..' segments resolve out of the store. id.ParseULIDLoose is applied only to FILENAMES on read (internal/storage/ticketpath.go:32), never to the frontmatter value.

Repro: kira init; create a ticket; hand-edit its frontmatter to 'id: ../../../ESCAPED'; run 'kira edit <n> --field title=pwned'. Result: ESCAPED.md is written outside the repository. The subsequent 'git add' fails ('outside repository') so nothing is committed, but WriteFileAtomic has already written the file -- the git error is not a guard, it is after the fact.

Impact: clone a repo containing a hostile ticket, run any kira mutation on it, and you get an attacker-controlled arbitrary file write (../../../.ssh/authorized_keys, shell rc, etc). Tickets travelling inside the repo is kira's core premise, so the hostile-repo path is the expected one, not exotic.

Fix at the upstream layer: reject a non-ULID 'id' in codec.Parse so a malformed item never becomes an Item. Add a defense-in-depth guard in storage.WriteItemRaw/ItemPath rejecting any ulid that is not ULID-shaped, since items also arrive from the merge driver and index rebuild. os.Root(KiraDir()) is a further option but is not required for the fix -- git only consumes path strings, so storage can keep producing them.

Found by the Go-modernization exploration (os.Root review), not by a security pass.
<!-- /kira:comment -->
