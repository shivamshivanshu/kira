package core

import (
	"slices"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/treeish"
)

func warningStrings(warns []error) []string {
	if len(warns) == 0 {
		return nil
	}
	out := make([]string, len(warns))
	for i, w := range warns {
		out[i] = w.Error()
	}
	return out
}

func warningsFromErrors(warns []error) []datamodel.Warning {
	return literalWarnings(warningStrings(warns))
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
