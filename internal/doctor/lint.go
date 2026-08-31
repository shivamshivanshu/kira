package doctor

import (
	"errors"
	"strconv"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
)

func Lint(content string) (it *datamodel.Item, parsed bool, findings []Finding) {
	it, err := codec.Parse(content)
	if err != nil {
		if pe, ok := errors.AsType[*datamodel.ParseError](err); ok {
			for _, e := range pe.Errs {
				findings = append(findings, Finding{Class: ClassSchema, Severity: SeverityError, Message: e.Error()})
			}
		} else {
			findings = append(findings, Finding{Class: ClassSchema, Severity: SeverityError, Message: err.Error()})
		}
	}
	if it != nil {
		findings = append(findings, commentFindings(it.Body)...)
	}
	return it, err == nil, findings
}

// unknownFindings warns per unknown key, except one the item's schema declares
// as a frontmatter extra and parsing captured into Extras — the same set
// core.guardWritable admits, so doctor warns exactly where writes block.
func unknownFindings(it *datamodel.Item, schemas map[string]entityschema.Schema) []Finding {
	out := make([]Finding, 0, len(it.UnknownKeys)+len(it.UnknownLinkTypes))
	add := func(field, kind string) {
		out = append(out, Finding{
			Class:    ClassSchema,
			Severity: SeverityWarning,
			Field:    field,
			Message:  "unknown " + kind + " " + strconv.Quote(field),
		})
	}
	for _, k := range it.UnknownKeys {
		_, declared := entityschema.DeclaredExtra(schemas, it.Type, k)
		if _, captured := it.Extras[k]; declared && captured {
			continue
		}
		add(k, "top-level field")
	}
	for _, t := range it.UnknownLinkTypes {
		add(t, "link type")
	}
	return out
}

func commentFindings(body string) []Finding {
	msgs := codec.LintComments(body)
	out := make([]Finding, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, Finding{Class: ClassComment, Severity: SeverityWarning, Message: m})
	}
	return out
}
