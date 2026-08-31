package gitx

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

func (r Repo) ShowCmd(sha string) *exec.Cmd {
	return gitCommand(r.Dir, nil, "show", sha)
}

const maxPatchBytes = 2 << 20

var ErrPatchTooLarge = errors.New("commit diff exceeds the 2 MiB preview limit")

func (r Repo) ShowPatch(ctx context.Context, sha string) (string, error) {
	cmd := gitCommandContext(ctx, r.Dir, nil, "--no-pager", "show", "--no-color", "--no-ext-diff", "--no-textconv", "--format=fuller", "--stat", "--patch", "--end-of-options", sha+"^{commit}", "--")
	var stdout patchBuffer
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if stdout.exceeded {
		return "", ErrPatchTooLarge
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		return "", cmdError("git show", &stderr, err)
	}
	return stdout.String(), nil
}

type patchBuffer struct {
	buffer   bytes.Buffer
	exceeded bool
}

func (b *patchBuffer) Write(p []byte) (int, error) {
	if len(p) > maxPatchBytes-b.buffer.Len() {
		b.exceeded = true
		return 0, ErrPatchTooLarge
	}
	return b.buffer.Write(p)
}

func (b *patchBuffer) String() string { return b.buffer.String() }
