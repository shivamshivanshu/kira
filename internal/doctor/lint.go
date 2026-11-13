package doctor

import (
	"errors"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
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

func commentFindings(body string) []Finding {
	msgs := codec.LintComments(body)
	out := make([]Finding, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, Finding{Class: ClassComment, Severity: SeverityWarning, Message: m})
	}
	return out
}
