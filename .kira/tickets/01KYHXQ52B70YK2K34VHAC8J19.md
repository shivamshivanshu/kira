---
id: 01KYHXQ52B70YK2K34VHAC8J19
number: CORE-47
aliases: []
type: ticket
title: "create --owner/--reporter @me is not resolved (unlike assign/edit/list)"
state: TODO
labels: []
epic: null
blocked_by: []
created: 2026-07-27T19:25:04+05:30
updated: 2026-07-27T19:25:04+05:30
---

`kira create ... --owner @me` (and presumably `--reporter @me`) stores the
literal string `@me` in the field instead of resolving it to the current git
user, unlike `assign`, `edit --owner`, `list --owner @me`, and
`list --query 'owner=@me'`, which all resolve it correctly.

Repro:

    kira create ticket --title X --no-edit --owner @me
    kira show KIRA-1 --json | grep owner     # "owner": "@me"
    kira list --owner @me                    # no items (real owner never set)

Visible symptom: a spurious "field \"owner\": value \"@me\" is not in the
\"person\" vocabulary" warning at create time.

`tests/e2e/testdata/script/me.txtar` covers assign/edit/list/board but not
create, which is why this slipped through.

Found while writing docs/user_manual/08-recipes-and-examples.md (kira
user-manual project) verifying a `create --owner @me` recipe.
