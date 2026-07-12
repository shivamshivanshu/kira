package core

import (
	"fmt"
	"os"
	"strings"
)

// invalidErr collapses a set of validation failures into one user error (exit
// 1) whose message lists every failure, mirroring item.ParseError's aggregation.
func invalidErr(errs []error) *Error {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return userErr("invalid item: %s", strings.Join(msgs, "; "))
}

// emitWarnings prints non-blocking validation warnings to stderr, keeping
// stdout free for --json output (docs/design/04-cli.md §7).
func emitWarnings(warns []error) {
	for _, w := range warns {
		fmt.Fprintln(os.Stderr, "kira: warning:", w.Error())
	}
}
