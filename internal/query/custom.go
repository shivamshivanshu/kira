package query

import (
	"slices"
	"strconv"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/entityschema"
)

// customField is a schema-declared field resolved across every loaded schema.
// A query spans item types, so the same name may be declared by more than one;
// declaringTypes records which, and an item of any other type simply does not
// match rather than making the whole query an error.
type customField struct {
	def            entityschema.FieldDef
	declaringTypes map[string]bool
}

func (c *compiler) lookupCustomField(name string) (customField, bool) {
	found := customField{declaringTypes: map[string]bool{}}
	for typ, schema := range c.opts.Schemas {
		if schema.ConfigBacked() {
			continue
		}
		f, ok := schema.Field(name)
		if !ok || !f.IsSchemaOnly() {
			continue
		}
		if found.def.Name == "" {
			found.def = f
		}
		found.declaringTypes[typ] = true
	}
	return found, found.def.Name != ""
}

func (c *compiler) customFieldNames() []string {
	var names []string
	for _, schema := range c.opts.Schemas {
		if schema.ConfigBacked() {
			continue
		}
		for _, f := range schema.Fields {
			if f.IsCustomFrontmatter() && !slices.Contains(names, f.Name) {
				names = append(names, f.Name)
			}
		}
	}
	return names
}

func (cf customField) declaredBy(it *datamodel.Item) bool { return cf.declaringTypes[it.Type] }

func (cf customField) values(it *datamodel.Item) []string {
	if !cf.declaredBy(it) {
		return nil
	}
	if cf.def.List {
		return it.ExtraLists[cf.def.Name]
	}
	if v, ok := it.ExtraScalars[cf.def.Name]; ok {
		return []string{v}
	}
	return nil
}

func (cf customField) scalar(it *datamodel.Item) string {
	if v := cf.values(it); len(v) == 1 {
		return v[0]
	}
	return ""
}

// compileCustomPred compares against a schema-declared field. An ordered
// comparison only means something where the declared type has an order, so on
// any other type it is rejected rather than silently compared as text.
func (c *compiler) compileCustomPred(n *predExpr, cf customField) (Predicate, error) {
	if !n.op.isOrderedCmp() {
		eq := n.op.kind == tokEq
		want := n.value
		return func(it *datamodel.Item, _ *datamodel.Config) bool {
			return slices.Contains(cf.values(it), want) == eq
		}, nil
	}
	switch cf.def.Type {
	case entityschema.FieldInt, entityschema.FieldNumber:
		want, err := strconv.ParseFloat(n.value, 64)
		if err != nil {
			return nil, &Error{Pos: n.valuePos, Msg: "invalid number " + quote(n.value)}
		}
		op := n.op.kind
		return func(it *datamodel.Item, _ *datamodel.Config) bool {
			got, err := strconv.ParseFloat(cf.scalar(it), 64)
			return err == nil && applyCmp(op, got, want)
		}, nil
	case entityschema.FieldDate, entityschema.FieldDatetime:
		want, err := parseDate(n.value)
		if err != nil {
			return nil, &Error{Pos: n.valuePos, Msg: "invalid date " + quote(n.value)}
		}
		n.date = want
		return datePred(n, cf.day), nil
	default:
		return nil, &Error{Pos: n.op.pos, Msg: "operator " + n.op.cmpText() + " is not valid on " + n.field +
			", declared " + string(cf.def.Type) + "; use = or !="}
	}
}

func (cf customField) day(it *datamodel.Item) string {
	parsed, err := parseDate(cf.scalar(it))
	if err != nil {
		return ""
	}
	return parsed.Format(time.DateOnly)
}

func (c *compiler) compileCustomIsEmpty(n *emptyExpr, cf customField) Predicate {
	notEmpty := n.notEmpty
	return func(it *datamodel.Item, _ *datamodel.Config) bool {
		return (len(cf.values(it)) > 0) == notEmpty
	}
}
