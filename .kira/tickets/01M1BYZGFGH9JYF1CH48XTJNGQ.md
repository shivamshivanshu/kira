---
id: 01M1BYZGFGH9JYF1CH48XTJNGQ
number: KIRA-126
aliases: []
type: ticket
title: "doctor flags the system captured label that kira create --here writes"
state: TODO
priority: P2
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:08:45+02:00
updated: 2026-08-31T15:08:45+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BYZGQD7FS92QX0V7H76Z74 author=Shivam-Shivanshu ts=2026-08-31T15:08:45+02:00 -->
doctor/check.go:92-95 checks every label against cfg.Labels.Known with no system-label exemption. The write path skips it (core/validate.go:23,60-64) and the extras path admits it (entityschema/project.go:141-142); explain.go:189 even documents it as excluded from vocabulary checks.
Scenario: labels.strict true plus any item created with kira create --here -> create succeeds and kira doctor then reports a vocab ERROR on a file kira itself wrote.
Fix: skip datamodel.CapturedLabel in vocabFindings, mirroring ConfigVocab.
<!-- /kira:comment -->
