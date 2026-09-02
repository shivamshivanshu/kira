---
id: 01M1HAA8BFF3WJ2453FFC8HJ9B
number: CORE-63
aliases: []
type: ticket
title: "Make container a declared capability instead of a hardcoded epic type"
state: DONE
resolution: done
priority: P2
labels: []
parent: null
blocked_by: []
created: 2026-09-02T17:03:04+02:00
updated: 2026-09-02T17:03:16+02:00
---
Nine sites across `internal/core` and `internal/cli` compared an item's type
against `datamodel.TypeEpic` to decide behaviour: progress rollup, tree
grouping, create-under-active defaulting, parent validity, and shell
completion. All nine encoded one idea -- "this type groups other items" --
which is exactly the pattern CORE-55 removed for display glyphs.

A type now declares `"container": true` in its schema, and `Schema.IsContainer`
plus `entityschema.ContainerTypes` turn that into the set the behaviour reads.
Only `epic` declares it, so a default repo behaves identically.

A parent ref cannot name its target type any more, because "any container" is
not a type name. `FieldDef` gained `targets_container`, and `TargetsItemType`
honours it. Emptying `target` instead was rejected after checking what depends
on it: `doctor.refFindings` only checks a schema field for dangling refs when
`TargetsItemType` holds, so an empty target would have silently stopped
reporting a dangling parent.

Containers may now nest, so `linkParent`'s refusal is gone -- part of CORE-59.

Two bugs found by the gate rather than by reading, both mine:
- `newValidationCtxWithEnums` builds the ctx as a struct literal and I threaded
  the new field into the wrong constructor, so the container set was always
  nil and every parent was rejected. A second constructor exists for Batch.
- `doctor.RefTypeMismatches` needed the same capability branch as
  `core.validateGraph`; without it, it compared against an empty target and
  flagged every parent in the repo -- 67 findings on this one.

The error text changed from "expected epic" to "does not group other items",
because naming one type is a lie once a feature can also be a parent.

Remaining `TypeEpic` uses are deliberate: `internal/seed` generates fixtures,
and `datamodel` declares the constant.
