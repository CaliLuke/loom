package testingx

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// ServicePackageShadowNames returns the identifiers that the generated Go
// files below dir declare where a service package imported under the same
// name would not compile, in the packages that import a service or views
// package, that is a package whose import path is genpkg/<name> or
// genpkg/<name>/views:
//
//   - the package-level declarations, which conflict with the import;
//   - the receivers, parameters, results and locals in scope where a
//     service or views package is used, which would hide it;
//   - in the files that do not import a service or views package, the
//     receivers, parameters, results and locals that are the operand of a
//     selector, such as ws in ws.Close(): a generated file keeps an import
//     when a selector of its name remains, so the file would keep the
//     service package import and leave it unused.
//
// It skips the service and views packages themselves. The result is sorted.
// It omits struct field names, names with upper case letters, which no
// generated package name has, and every name that contains marker, so that
// a test design can mark the names it gives to generated code.
func ServicePackageShadowNames(t testing.TB, dir, genpkg, marker string) []string {
	t.Helper()
	marker = strings.ToLower(marker)
	names := make(map[string]struct{})
	packages := make(map[string][]*ast.File)
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if err != nil {
			return err
		}
		packages[filepath.Dir(p)] = append(packages[filepath.Dir(p)], file)
		return nil
	})
	if err != nil {
		t.Fatalf("parse generated files: %v", err)
	}
	add := func(name string) {
		if name != "_" && strings.ToLower(name) == name && !strings.Contains(name, marker) {
			names[name] = struct{}{}
		}
	}
	for pkgDir, files := range packages {
		if isServicePackageDir(dir, pkgDir, genpkg) {
			continue
		}
		if !slices.ContainsFunc(files, func(file *ast.File) bool { return len(serviceImportNames(file, genpkg)) > 0 }) {
			continue
		}
		for _, file := range files {
			for _, name := range packageLevelNames(file) {
				add(name)
			}
			services := serviceImportNames(file, genpkg)
			v := &scopeVisitor{packages: services, local: len(services) == 0, add: add}
			for _, decl := range file.Decls {
				ast.Walk(v, decl)
			}
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}

// isServicePackageDir reports whether pkgDir, a directory below the module
// directory dir, holds a service or views package of genpkg. Those packages
// do not import a service package under a transport alias.
func isServicePackageDir(dir, pkgDir, genpkg string) bool {
	rel, err := filepath.Rel(dir, pkgDir)
	if err != nil {
		return false
	}
	elems := strings.Split(filepath.ToSlash(rel), "/")
	if elems[0] != path.Base(genpkg) {
		return false
	}
	return len(elems) == 2 || len(elems) == 3 && elems[2] == "views"
}

// serviceImportNames returns the names under which file imports the service
// and views packages of genpkg.
func serviceImportNames(file *ast.File, genpkg string) map[string]bool {
	names := make(map[string]bool)
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		rel, ok := strings.CutPrefix(importPath, genpkg+"/")
		if !ok {
			continue
		}
		elems := strings.Split(rel, "/")
		if len(elems) > 2 || len(elems) == 2 && elems[1] != "views" {
			continue
		}
		name := path.Base(importPath)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		names[name] = true
	}
	return names
}

// scopeVisitor walks a declaration and reports the names in scope where a
// selector uses one of packages, and, when local is set, the declared names
// that are the operand of a selector.
type scopeVisitor struct {
	parent   *scopeVisitor
	names    []string
	packages map[string]bool
	local    bool
	add      func(string)
}

// Visit implements ast.Visitor.
func (v *scopeVisitor) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case *ast.StructType, *ast.InterfaceType:
		return nil
	case *ast.BlockStmt, *ast.IfStmt, *ast.ForStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.CaseClause, *ast.CommClause, *ast.SelectStmt:
		return v.child()
	case *ast.FuncDecl:
		v.visitFunc(node.Recv, node.Type, node.Body)
		return nil
	case *ast.FuncLit:
		v.visitFunc(nil, node.Type, node.Body)
		return nil
	case *ast.RangeStmt:
		v.visitRange(node)
		return nil
	case *ast.AssignStmt:
		if node.Tok != token.DEFINE {
			return v
		}
		v.declare(node.Lhs, node.Rhs...)
		return nil
	case *ast.ValueSpec:
		names := make([]ast.Expr, len(node.Names))
		for i, id := range node.Names {
			names[i] = id
		}
		if node.Type != nil {
			ast.Walk(v, node.Type)
		}
		v.declare(names, node.Values...)
		return nil
	case *ast.SelectorExpr:
		v.visitSelector(node)
	}
	return v
}

// visitFunc walks body in a scope that declares the receiver, parameters
// and results of a function.
func (v *scopeVisitor) visitFunc(recv *ast.FieldList, typ *ast.FuncType, body *ast.BlockStmt) {
	child := v.child()
	child.fields(recv, typ.Params, typ.Results)
	if body != nil {
		ast.Walk(child, body)
	}
}

// visitRange walks the range expression of node in the scope of v and its
// body in a scope that declares its key and value.
func (v *scopeVisitor) visitRange(node *ast.RangeStmt) {
	ast.Walk(v, node.X)
	child := v.child()
	if node.Tok == token.DEFINE {
		child.idents(node.Key, node.Value)
	}
	ast.Walk(child, node.Body)
}

// declare walks values, which are in scope before the declaration, then
// declares the identifiers among names.
func (v *scopeVisitor) declare(names []ast.Expr, values ...ast.Expr) {
	for _, value := range values {
		ast.Walk(v, value)
	}
	v.idents(names...)
}

// visitSelector reports the names in scope when node selects from one of
// the service packages, and the operand of node when it is a declared name
// and v is local.
func (v *scopeVisitor) visitSelector(node *ast.SelectorExpr) {
	id, ok := node.X.(*ast.Ident)
	if !ok {
		return
	}
	if v.packages[id.Name] {
		for s := v; s != nil; s = s.parent {
			for _, name := range s.names {
				v.add(name)
			}
		}
	}
	if v.local && v.declares(id.Name) {
		v.add(id.Name)
	}
}

// child returns a visitor for a scope nested in the scope of v.
func (v *scopeVisitor) child() *scopeVisitor {
	return &scopeVisitor{parent: v, packages: v.packages, local: v.local, add: v.add}
}

// fields declares the names of the field lists in the scope of v.
func (v *scopeVisitor) fields(lists ...*ast.FieldList) {
	for _, list := range lists {
		if list == nil {
			continue
		}
		for _, field := range list.List {
			for _, id := range field.Names {
				v.names = append(v.names, id.Name)
			}
		}
	}
}

// idents declares the identifiers among exprs in the scope of v.
func (v *scopeVisitor) idents(exprs ...ast.Expr) {
	for _, expr := range exprs {
		if id, ok := expr.(*ast.Ident); ok {
			v.names = append(v.names, id.Name)
		}
	}
}

// declares reports whether name is declared in the scope of v.
func (v *scopeVisitor) declares(name string) bool {
	for s := v; s != nil; s = s.parent {
		if slices.Contains(s.names, name) {
			return true
		}
	}
	return false
}

// packageLevelNames returns the names of the package-level declarations of
// file.
func packageLevelNames(file *ast.File) []string {
	var names []string
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if decl.Recv == nil {
				names = append(names, decl.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					for _, id := range spec.Names {
						names = append(names, id.Name)
					}
				case *ast.TypeSpec:
					names = append(names, spec.Name.Name)
				}
			}
		}
	}
	return names
}
