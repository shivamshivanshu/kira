package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// editorEnv returns the command to run for $EDITOR, or an error (exit 3) when
// it is unset — a required-but-missing environment dependency.
func editorEnv() ([]string, error) {
	ed := strings.TrimSpace(os.Getenv("EDITOR"))
	if ed == "" {
		ed = strings.TrimSpace(os.Getenv("VISUAL"))
	}
	if ed == "" {
		return nil, envErr("$EDITOR is not set")
	}
	return strings.Fields(ed), nil
}

// runEditor drives the parse-validate-retry loop (docs/design/04-cli.md §6):
// it writes initial to a temp file, opens $EDITOR on it, and re-validates on
// save. On failure it re-opens with the errors prepended as an HTML-comment
// banner and loops, until the document is valid or the user saves the
// error-annotated buffer unchanged (an abort). validate returns the hard errors
// for a given buffer, or nil when it is acceptable.
func runEditor(initial string, validate func(content string) []error) (string, error) {
	editor, err := editorEnv()
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp("", "kira-*.md")
	if err != nil {
		return "", userErr("creating editor buffer: %v", err)
	}
	path := tmp.Name()
	tmp.Close()
	defer os.Remove(path)

	buffer := initial
	annotated := false
	for {
		if err := os.WriteFile(path, []byte(buffer), 0o600); err != nil {
			return "", userErr("writing editor buffer: %v", err)
		}
		if err := spawnEditor(editor, path); err != nil {
			return "", err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", userErr("reading editor buffer: %v", err)
		}
		edited := string(raw)
		content := stripErrorBanner(edited)
		errs := validate(content)
		if len(errs) == 0 {
			return content, nil
		}
		// The user saved the error-annotated buffer without changing it: treat
		// as an abort rather than looping forever.
		if annotated && edited == buffer {
			return "", userErr("aborted: %d validation error(s), buffer unchanged", len(errs))
		}
		buffer = errorBanner(errs) + content
		annotated = true
	}
}

// spawnEditor runs the editor command against path with the terminal attached.
func spawnEditor(editor []string, path string) error {
	args := append(slices.Clone(editor[1:]), path)
	cmd := exec.Command(editor[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return userErr("editor %s: %v", filepath.Base(editor[0]), err)
	}
	return nil
}
