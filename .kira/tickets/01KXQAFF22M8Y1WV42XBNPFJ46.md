---
id: 01KXQAFF22M8Y1WV42XBNPFJ46
number: INFRA-2
aliases: []
type: ticket
subtype: task
title: "Package prebuilt release binaries via GoReleaser"
state: DONE
resolution: done
priority: P2
owner: shivam
reporter: shivam
labels: []
epic: null
blocked_by: []
created: 2026-07-17T11:28:34+05:30
updated: 2026-07-17T11:28:41+05:30
---

## Description

Users previously had to `git clone` and `go build` kira themselves. Added
GoReleaser-based packaging so tagged releases publish prebuilt binaries
to GitHub Releases directly.

- `.goreleaser.yaml`: builds `cmd/kira` for linux/darwin x amd64/arm64.
  - `CGO_ENABLED=0`: project has no cgo dependencies (verified), so the
    binary is statically linked — avoids `GLIBC_x.xx not found` failures
    on user machines with an older glibc than the CI runner's.
  - `-trimpath`: strips CI runner's absolute build paths from the binary
    for reproducibility.
  - `-ldflags "-s -w"`: strips symbol table/DWARF debug info (smaller
    binary); also injects `-X internal/cli.version={{.Version}}` into
    the existing ldflags-overridable `version` var.
  - Archives as `.tar.gz` per OS/arch, plus a `checksums.txt`.
  - No Windows target (not requested).
- `.github/workflows/ci.yml`: added a `release` job, gated on the
  existing `tag` job.
  - Gotcha: the `tag` job pushes its tag using the default
    `GITHUB_TOKEN`, and GitHub's anti-recursion rule means pushes made
    with that token do not trigger other workflow runs. A separate
    `release.yml` on `push: tags` would silently never fire. Fixed by
    making `release` a job in the same workflow, depending on `tag` and
    consuming the tag name via a job output (`tag` job's `next` value)
    instead of relying on a tag-push trigger.
  - Runs `goreleaser/goreleaser-action@v6` pinned to `v2.5.1`.

Go has no `-O2`/`-O3` equivalent (the `gc` compiler always builds at one
fixed, always-on optimization level); `-N -l` only *disables*
optimization/inlining for debugger builds. Considered and declined:
`GOAMD64=v2/v3` (risks illegal-instruction on older CPUs with no data on
target hardware) and PGO (needs a representative profile not yet
collected).

Verified locally with `goreleaser release --snapshot --clean`: produced
4 archives (linux/darwin x amd64/arm64), confirmed the binary is
statically linked (`ldd` reports "not a dynamic executable") and
`kira version` reports the injected version string.

## Acceptance criteria

- Tagged pushes to main produce linux/darwin (amd64/arm64) release
  archives + checksums on GitHub Releases, with no manual build step
  for users.

## Comments
