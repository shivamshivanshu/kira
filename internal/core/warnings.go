package core

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/treeish"
)

func emitWarnings(warns []error) {
	for _, w := range warns {
		fmt.Fprintln(os.Stderr, "kira: warning:", w.Error())
		var ce *errx.Error
		if errors.As(w, &ce) && ce.Hint != "" {
			fmt.Fprintln(os.Stderr, "  hint:", ce.Hint)
		}
	}
}

func literalWarnings(msgs []string) []datamodel.Warning {
	if len(msgs) == 0 {
		return nil
	}
	out := make([]datamodel.Warning, len(msgs))
	for i, m := range msgs {
		out[i] = datamodel.Warning{Code: datamodel.WarnLiteral, Args: []string{m}}
	}
	return out
}

func mergedWarnings(a, b *treeish.Loaded) []datamodel.Warning {
	return literalWarnings(slices.Concat(a.Warnings, b.Warnings))
}
