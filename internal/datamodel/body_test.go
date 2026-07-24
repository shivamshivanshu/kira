package datamodel

import "testing"

func TestUpsertBodySectionInsertsBeforeTrailingComments(t *testing.T) {
	body := "\n## Description\n\nsome prose\n\n## Comments\n"

	got := UpsertBodySection(body, "Notes", "a note")

	want := "\n## Description\n\nsome prose\n\n## Notes\n\na note\n\n## Comments\n"
	if got != want {
		t.Fatalf("Notes must be inserted before Comments\ngot:  %q\nwant: %q", got, want)
	}
}

func TestUpsertBodySectionAppendsAtEndWhenNoComments(t *testing.T) {
	body := "\n## Description\n\nsome prose\n"

	got := UpsertBodySection(body, "Notes", "a note")

	want := "\n## Description\n\nsome prose\n\n## Notes\n\na note\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestUpsertBodySectionEmptyContentRemovesSection(t *testing.T) {
	body := "\n## Description\n\nsome prose\n\n## Notes\n\na note\n\n## Comments\n"

	got := UpsertBodySection(body, "Notes", "")

	want := "\n## Description\n\nsome prose\n\n## Comments\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestUpsertBodySectionEmptyContentOnMissingSectionIsNoop(t *testing.T) {
	body := "\n## Description\n\nsome prose\n"

	if got := UpsertBodySection(body, "Notes", ""); got != body {
		t.Fatalf("clearing a section that never existed must be a no-op, got %q", got)
	}
}

func TestUpsertBodySectionRemovingLastSectionLeavesNoTrailingBlank(t *testing.T) {
	body := "\n## Description\n\nprose\n\n## Notes\n\na note\n"

	got := UpsertBodySection(body, "Notes", "")

	want := "\n## Description\n\nprose\n"
	if got != want {
		t.Fatalf("removing the last section must not leave a dangling blank\ngot:  %q\nwant: %q", got, want)
	}
}

func TestUpsertBodySectionRemovingOnlySectionYieldsEmpty(t *testing.T) {
	if got := UpsertBodySection("## Notes\n\na note\n", "Notes", ""); got != "" {
		t.Fatalf("removing the only section must yield an empty body, got %q", got)
	}
}

func TestContainsSectionHeading(t *testing.T) {
	if !ContainsSectionHeading("intro\n## Overview\nmore") {
		t.Error("a line starting with \"## \" must be detected")
	}
	if ContainsSectionHeading("### sub\nplain text") {
		t.Error("a deeper heading or plain text must not count as a section heading")
	}
}
