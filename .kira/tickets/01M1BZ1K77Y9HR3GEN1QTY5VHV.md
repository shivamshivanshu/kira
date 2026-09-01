---
id: 01M1BZ1K77Y9HR3GEN1QTY5VHV
number: KIRA-139
aliases: []
type: ticket
title: "Correct the package-layering documentation to match reality"
state: DONE
resolution: done
priority: P3
labels: []
epic: null
blocked_by: []
created: 2026-08-31T15:09:54+02:00
updated: 2026-09-01T10:25:59+02:00
---

## Description

## Acceptance criteria

## Comments

<!-- kira:comment id=01M1BZ1KFM45K9546N2C895B3H author=Shivam-Shivanshu ts=2026-08-31T15:09:54+02:00 -->
From the data contextual DX-1 review. go list shows: datamodel imports errx, ptr, setx and timex, while CLAUDE.md says 'depends only on the setx leaf' (already amended once this session for setx, still wrong for the rest). treeish imports config, while CLAUDE.md lists config many layers above treeish; the real order is config -> query, storage and treeish -> config, with no cycle.
Fix: correct CLAUDE.md and REVIEW.md to state the actual tiers, or invert treeish's config dependency by injecting a parsed config. The doc fix is the honest minimum - the layering itself is sound.
<!-- /kira:comment -->
