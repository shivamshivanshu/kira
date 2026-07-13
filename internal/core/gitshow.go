package core

import "os/exec"

func (s *Store) CommitShowCmd(sha string) *exec.Cmd { return s.repo().ShowCmd(sha) }
