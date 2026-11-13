package gitx

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"
)

// remoteTimeout bounds Pull and Push so a stalled remote cannot wedge
// callers that hold locks across a sync, like the web dashboard.
var remoteTimeout = 10 * time.Minute

func (r Repo) Pull(remote string) error {
	args := []string{"pull", "--rebase"}
	if remote != "" {
		args = append(args, remote)
	}
	return r.runRemote(remote, args)
}

func (r Repo) Push(remote string) error {
	args := []string{"push"}
	if remote != "" {
		args = append(args, remote)
	}
	return r.runRemote(remote, args)
}

func (r Repo) runRemote(remote string, args []string) error {
	if err := requireNonFlag("remote", remote); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), remoteTimeout)
	defer cancel()
	cmd := gitCommandContext(ctx, r.Dir, nil, args...)
	// The transport child (ssh) inherits the output pipes; without WaitDelay
	// a killed git would leave Run blocked until that child exits too.
	cmd.WaitDelay = 5 * time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil && ctx.Err() != nil {
		return fmt.Errorf("git %s: timed out after %s", args[0], remoteTimeout)
	}
	if err != nil {
		return cmdError("git "+strings.Join(args, " "), &stderr, err)
	}
	return nil
}

func (r Repo) RebaseContinue() error {
	_, err := r.outputRaw([]string{"GIT_EDITOR=true"}, "rebase", "--continue")
	return err
}

func (r Repo) RebaseSkip() error {
	_, err := r.Output("rebase", "--skip")
	return err
}

func (r Repo) RebaseAbort() error {
	_, err := r.Output("rebase", "--abort")
	return err
}
