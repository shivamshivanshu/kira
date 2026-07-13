package config

import (
	"strings"
	"testing"
)

const setFixture = `version: 1

project:
  key: KIRA
  name: kira

commit:
  mode: auto                  # auto | manual | prompt
  trailer: Kira-Ticket
  close_trailer: Kira-Closes

sync:
  push: false                 # true | false

workon:
  branch_pattern: "{key}/{number}-{slug}"   # tokens: {key} {number} {slug}
  casing: kebab               # kebab | snake

estimate:
  unit: points                # points | hours

# git: relate items to commits. docs: docs/design/07-git-integration.md
#git: {}
`

func changedLines(a, b string) []int {
	al, bl := strings.Split(a, "\n"), strings.Split(b, "\n")
	var diff []int
	for i := 0; i < len(al) || i < len(bl); i++ {
		var x, y string
		if i < len(al) {
			x = al[i]
		}
		if i < len(bl) {
			y = bl[i]
		}
		if x != y {
			diff = append(diff, i)
		}
	}
	return diff
}

func TestSetScalarPreservesEveryOtherLine(t *testing.T) {
	cases := []struct {
		key, value, wantLine string
	}{
		{"commit.mode", "manual", "  mode: manual                  # auto | manual | prompt"},
		{"sync.push", "true", "  push: true                 # true | false"},
		{"workon.branch_pattern", "kira/{number}", "  branch_pattern: kira/{number}   # tokens: {key} {number} {slug}"},
		{"project.name", "my proj", "  name: my proj"},
	}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			out, err := SetScalar([]byte(setFixture), c.key, c.value)
			if err != nil {
				t.Fatalf("SetScalar: %v", err)
			}
			diff := changedLines(setFixture, string(out))
			if len(diff) != 1 {
				t.Fatalf("changed %d lines, want 1:\n%s", len(diff), out)
			}
			got := strings.Split(string(out), "\n")[diff[0]]
			if got != c.wantLine {
				t.Errorf("changed line =\n%q\nwant\n%q", got, c.wantLine)
			}
		})
	}
}

func TestSetScalarCreatesAbsentBlock(t *testing.T) {
	out, err := SetScalar([]byte(setFixture), "git.landed_ref", "origin/main")
	if err != nil {
		t.Fatalf("SetScalar: %v", err)
	}
	cfg, err := Parse(out)
	if err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if cfg.Git.LandedRef != "origin/main" {
		t.Errorf("git.landed_ref = %q, want origin/main", cfg.Git.LandedRef)
	}
	if !strings.Contains(string(out), "#git: {}") {
		t.Errorf("commented git doc block was lost:\n%s", out)
	}
	// setting it again is a plain edit, not a second block
	out2, err := SetScalar(out, "git.landed_ref", "origin/master")
	if err != nil {
		t.Fatalf("second SetScalar: %v", err)
	}
	if n := strings.Count(string(out2), "\nlanded_ref:") + strings.Count(string(out2), "\n  landed_ref:"); n != 1 {
		t.Errorf("landed_ref appears %d times, want 1:\n%s", n, out2)
	}
}

func TestSetScalarInsertsLeafUnderExistingParent(t *testing.T) {
	// an older config whose commit block predates close_trailer
	src := "version: 1\n\ncommit:\n  mode: auto                  # auto | manual | prompt\n  trailer: Kira-Ticket\n"
	out, err := SetScalar([]byte(src), "commit.close_trailer", "Kira-Closes")
	if err != nil {
		t.Fatalf("SetScalar: %v", err)
	}
	if !strings.Contains(string(out), "\n  close_trailer: Kira-Closes\n") {
		t.Errorf("close_trailer not inserted under commit at the right indent:\n%s", out)
	}
	cfg, err := Parse(out)
	if err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if cfg.Commit.CloseTrailer != "Kira-Closes" {
		t.Errorf("commit.close_trailer = %q, want Kira-Closes", cfg.Commit.CloseTrailer)
	}
}

func TestSetScalarRejects(t *testing.T) {
	if _, err := SetScalar([]byte(setFixture), "nope.here", "x"); err == nil {
		t.Error("unknown key accepted")
	} else if !strings.Contains(err.Error(), "commit.mode") {
		t.Errorf("unknown-key error should list valid keys, got: %v", err)
	}
	if _, err := SetScalar([]byte(setFixture), "commit.mode", "bogus"); err == nil {
		t.Error("invalid enum value accepted")
	}
	if _, err := SetScalar([]byte(setFixture), "sync.push", "maybe"); err == nil {
		t.Error("invalid bool value accepted")
	}
}
