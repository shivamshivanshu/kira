package schema

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shivamshivanshu/kira/internal/automation"
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

// TestReachableStringEnumsRegistered guards against an enum-shaped named
// string type shipping as a bare "type": "string" — the gap DATA-10 fixed.
// It AST-scans the packages topLevelTypes() spans for named string types
// backed by a const block, and requires every such type reachable from
// topLevelTypes() to be present in stringEnums.
func TestReachableStringEnumsRegistered(t *testing.T) {
	pkgDirs := map[string]string{
		reflect.TypeFor[datamodel.CreateResult]().PkgPath(): "../datamodel",
		reflect.TypeFor[doctor.Report]().PkgPath():          "../doctor",
		reflect.TypeFor[syncx.Report]().PkgPath():           "../syncx",
		reflect.TypeFor[automation.HookPayload]().PkgPath(): "../automation",
	}

	constBacked := map[string]bool{}
	for pkgPath, dir := range pkgDirs {
		files, err := filepath.Glob(dir + "/*.go")
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		fset := token.NewFileSet()
		stringTypes := map[string]bool{}
		constTyped := map[string]bool{}
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for _, decl := range file.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				switch gd.Tok {
				case token.TYPE:
					for _, spec := range gd.Specs {
						ts, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}
						if ident, ok := ts.Type.(*ast.Ident); ok && ident.Name == "string" {
							stringTypes[ts.Name.Name] = true
						}
					}
				case token.CONST:
					for _, spec := range gd.Specs {
						vs, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						if ident, ok := vs.Type.(*ast.Ident); ok {
							constTyped[ident.Name] = true
						}
					}
				}
			}
		}
		for name := range stringTypes {
			if constTyped[name] {
				constBacked[pkgPath+"."+name] = true
			}
		}
	}

	reachable := map[reflect.Type]bool{}
	seen := map[reflect.Type]bool{}
	for _, rt := range topLevelTypes() {
		collectNamedStringTypes(rt, seen, reachable)
	}

	for rt := range reachable {
		if _, ok := pkgDirs[rt.PkgPath()]; !ok {
			t.Errorf("%s is reachable from topLevelTypes() but its package is not scanned by this test; add it to pkgDirs", rt)
			continue
		}
		if !constBacked[rt.PkgPath()+"."+rt.Name()] {
			continue
		}
		if _, ok := stringEnums[rt]; !ok {
			t.Errorf("%s is const-backed and reachable from topLevelTypes() but missing from stringEnums", rt)
		}
	}
}

func collectNamedStringTypes(t reflect.Type, seen, out map[reflect.Type]bool) {
	if seen[t] {
		return
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.String:
		if t.PkgPath() != "" {
			out[t] = true
		}
	case reflect.Pointer, reflect.Slice, reflect.Array:
		collectNamedStringTypes(t.Elem(), seen, out)
	case reflect.Map:
		collectNamedStringTypes(t.Key(), seen, out)
		collectNamedStringTypes(t.Elem(), seen, out)
	case reflect.Struct:
		for i := range t.NumField() {
			f := t.Field(i)
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if !f.IsExported() || name == "-" {
				continue
			}
			collectNamedStringTypes(f.Type, seen, out)
		}
	}
}
