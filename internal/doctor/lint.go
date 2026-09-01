package doctor

import (
	"errors"
	"maps"
	"slices"
	"strconv"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
)

func Lint(content string, schemas map[string]entityschema.Schema) (it *datamodel.Item, parsed bool, findings []Finding) {
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
		findings = append(findings, unknownFindings(it, schemas[it.Type])...)
		findings = append(findings, commentFindings(it.Body)...)
	}
	return it, err == nil, findings
}

// unknownFindings warns on a top-level extra field only when the item's type
// schema doesn't declare it; a schema-declared extra is legitimate, not
// unknown.
func unknownFindings(it *datamodel.Item, schema entityschema.Schema) []Finding {
	extraKeys := it.ExtraKeys()
	out := make([]Finding, 0, len(extraKeys)+len(it.ExtraLinks))
	add := func(field, kind string) {
		out = append(out, Finding{
			Class:    ClassSchema,
			Severity: SeverityWarning,
			Field:    field,
			Message:  "unknown " + kind + " " + strconv.Quote(field),
		})
	}
	for _, k := range extraKeys {
		if _, declared := schema.Field(k); declared {
			continue
		}
		add(k, "top-level field")
	}
	for _, t := range slices.Sorted(maps.Keys(it.ExtraLinks)) {
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
