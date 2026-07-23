package schema

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/doctor"
	"github.com/shivamshivanshu/kira/internal/syncx"
)

// TestAllDatamodelResultTypesRegistered guards against a *Result type being
// added to datamodel and never wired into topLevelTypes() — the bug this
// ticket exists to fix. It parses the datamodel package source directly
// (rather than reflecting over an instance) since there is no runtime API
// to enumerate a package's declared types.
func TestAllDatamodelResultTypesRegistered(t *testing.T) {
	files, err := filepath.Glob("../datamodel/*.go")
	if err != nil {
		t.Fatalf("glob datamodel package: %v", err)
	}

	fset := token.NewFileSet()
	declared := map[string]bool{}
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if _, ok := ts.Type.(*ast.StructType); !ok {
					continue
				}
				if ts.Name.IsExported() && strings.HasSuffix(ts.Name.Name, "Result") {
					declared[ts.Name.Name] = true
				}
			}
		}
	}

	datamodelPkgPath := reflect.TypeFor[datamodel.CreateResult]().PkgPath()
	registered := map[string]bool{}
	for _, rt := range topLevelTypes() {
		if rt.PkgPath() == datamodelPkgPath {
			registered[rt.Name()] = true
		}
	}

	for name := range declared {
		if !registered[name] {
			t.Errorf("datamodel.%s is declared but not registered in topLevelTypes()", name)
		}
	}
}

// TestOtherReportTypesRegistered covers the non-datamodel result-producing
// types (doctor and syncx reports) that the AST scan above doesn't reach.
func TestOtherReportTypesRegistered(t *testing.T) {
	want := []reflect.Type{
		reflect.TypeFor[doctor.Report](),
		reflect.TypeFor[syncx.Report](),
	}
	registered := map[reflect.Type]bool{}
	for _, rt := range topLevelTypes() {
		registered[rt] = true
	}
	for _, rt := range want {
		if !registered[rt] {
			t.Errorf("%s is not registered in topLevelTypes()", rt)
		}
	}
}
