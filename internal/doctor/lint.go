package doctor

import (
	"errors"
	"slices"
	"strconv"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

var knownFrontmatterKeys = func() map[string]bool {
	m := make(map[string]bool, len(datamodel.FrontmatterKeys))
	for _, k := range datamodel.FrontmatterKeys {
		m[k] = true
	}
	return m
}()

func Lint(content string) (it *datamodel.Item, parsed bool, findings []Finding) {
	it, keys, err := codec.ParseKeys(content)
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
	findings = append(findings, unknownFieldFindings(keys)...)
	if it != nil {
		findings = append(findings, commentFindings(it.Body)...)
	}
	return it, err == nil, findings
}

func unknownFieldFindings(keys []string) []Finding {
	var unknown []string
	for _, k := range keys {
		if !knownFrontmatterKeys[k] {
			unknown = append(unknown, k)
		}
	}
	slices.Sort(unknown)
	out := make([]Finding, 0, len(unknown))
	for _, k := range unknown {
		out = append(out, Finding{
			Class:    ClassSchema,
			Severity: SeverityWarning,
			Field:    k,
			Message:  "unknown top-level field " + strconv.Quote(k),
		})
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
