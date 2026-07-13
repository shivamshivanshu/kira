package doctor

import (
	"errors"
	"strconv"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func Lint(content string) (it *datamodel.Item, parsed bool, findings []Finding) {
	it, err := codec.Parse(content)
	if err != nil {
		var pe *datamodel.ParseError
		if errors.As(err, &pe) {
			for _, e := range pe.Errs {
				findings = append(findings, Finding{Class: ClassSchema, Severity: SeverityError, Message: e.Error()})
			}
		} else {
			findings = append(findings, Finding{Class: ClassSchema, Severity: SeverityError, Message: err.Error()})
		}
	}
	if it != nil {
		findings = append(findings, unknownFindings(it)...)
		findings = append(findings, commentFindings(it.Body)...)
	}
	return it, err == nil, findings
}

func unknownFindings(it *datamodel.Item) []Finding {
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
